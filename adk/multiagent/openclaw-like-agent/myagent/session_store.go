package myagent

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type sessionMeta struct {
	SessionID string    `json:"session_id"`
	Summary   string    `json:"summary,omitempty"`
	Skip      int       `json:"skip,omitempty"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type jsonlMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	ToolName   string            `json:"tool_name,omitempty"`
	ToolCalls  []schema.ToolCall `json:"tool_calls,omitempty"`
}

type jsonlSessionStore struct {
	root string
}

func newJSONLSessionStore(root string) (*jsonlSessionStore, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("创建 sessions 目录失败: %w", err)
	}
	return &jsonlSessionStore{root: root}, nil
}

func (s *jsonlSessionStore) EnsureSession(sessionID string) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	if _, err := os.Stat(s.metaPath(sessionID)); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查 session meta 失败: %w", err)
	}
	now := time.Now().UTC()
	return s.writeMeta(sessionID, sessionMeta{
		SessionID: sessionID,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *jsonlSessionStore) AddMessage(sessionID string, msg *schema.Message) error {
	return s.AddMessages(sessionID, msg)
}

func (s *jsonlSessionStore) AddMessages(sessionID string, messages ...*schema.Message) error {
	if err := s.EnsureSession(sessionID); err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}

	file, err := os.OpenFile(s.jsonlPath(sessionID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("打开 session jsonl 失败: %w", err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	count := 0
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		row := jsonlMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			ToolName:   msg.ToolName,
			ToolCalls:  append([]schema.ToolCall(nil), msg.ToolCalls...),
		}
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("写入 session message 失败: %w", err)
		}
		count++
	}

	meta, err := s.readMeta(sessionID)
	if err != nil {
		return err
	}
	meta.Count += count
	meta.UpdatedAt = time.Now().UTC()
	return s.writeMeta(sessionID, meta)
}

func (s *jsonlSessionStore) GetHistory(sessionID string) ([]*schema.Message, error) {
	if err := s.EnsureSession(sessionID); err != nil {
		return nil, err
	}
	file, err := os.Open(s.jsonlPath(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("打开 session jsonl 失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var history []*schema.Message
	for scanner.Scan() {
		var row jsonlMessage
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, fmt.Errorf("解析 session jsonl 失败: %w", err)
		}
		history = append(history, &schema.Message{
			Role:       schema.RoleType(row.Role),
			Content:    row.Content,
			ToolCallID: row.ToolCallID,
			ToolName:   row.ToolName,
			ToolCalls:  append([]schema.ToolCall(nil), row.ToolCalls...),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 session jsonl 失败: %w", err)
	}
	return history, nil
}

func (s *jsonlSessionStore) TouchSummary(sessionID, assistantReply string) error {
	meta, err := s.readMeta(sessionID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(meta.Summary) == "" && strings.TrimSpace(assistantReply) != "" {
		meta.Summary = trimForDisplay(assistantReply, 120)
	}
	meta.UpdatedAt = time.Now().UTC()
	return s.writeMeta(sessionID, meta)
}

func (s *jsonlSessionStore) ClearSession(sessionID string) error {
	if err := s.EnsureSession(sessionID); err != nil {
		return err
	}
	if err := os.WriteFile(s.jsonlPath(sessionID), nil, 0o644); err != nil {
		return fmt.Errorf("清空 session jsonl 失败: %w", err)
	}
	meta, err := s.readMeta(sessionID)
	if err != nil {
		return err
	}
	meta.Summary = ""
	meta.Skip = 0
	meta.Count = 0
	meta.UpdatedAt = time.Now().UTC()
	return s.writeMeta(sessionID, meta)
}

func (s *jsonlSessionStore) DeleteSession(sessionID string) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	if err := os.Remove(s.jsonlPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 session jsonl 失败: %w", err)
	}
	if err := os.Remove(s.metaPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 session meta 失败: %w", err)
	}
	return nil
}

func (s *jsonlSessionStore) ListSessions() ([]sessionMeta, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("读取 sessions 目录失败: %w", err)
	}
	var metas []sessionMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".meta.json") {
			continue
		}
		sessionID := strings.TrimSuffix(entry.Name(), ".meta.json")
		meta, err := s.readMeta(sessionID)
		if err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].UpdatedAt.After(metas[j].UpdatedAt)
	})
	return metas, nil
}

func (s *jsonlSessionStore) readMeta(sessionID string) (sessionMeta, error) {
	data, err := os.ReadFile(s.metaPath(sessionID))
	if err != nil {
		return sessionMeta{}, fmt.Errorf("读取 session meta 失败: %w", err)
	}
	var meta sessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return sessionMeta{}, fmt.Errorf("解析 session meta 失败: %w", err)
	}
	return meta, nil
}

func (s *jsonlSessionStore) writeMeta(sessionID string, meta sessionMeta) error {
	payload, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 session meta 失败: %w", err)
	}
	if err := os.WriteFile(s.metaPath(sessionID), payload, 0o644); err != nil {
		return fmt.Errorf("写入 session meta 失败: %w", err)
	}
	return nil
}

func (s *jsonlSessionStore) jsonlPath(sessionID string) string {
	return filepath.Join(s.root, sessionID+".jsonl")
}

func (s *jsonlSessionStore) metaPath(sessionID string) string {
	return filepath.Join(s.root, sessionID+".meta.json")
}

func validateSessionID(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id 不能为空")
	}
	if !sessionIDPattern.MatchString(sessionID) {
		return fmt.Errorf("非法 session id %q，仅支持字母、数字、下划线和中划线", sessionID)
	}
	return nil
}

func newSessionID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}
	return "sess_" + hex.EncodeToString(buf[:])
}
