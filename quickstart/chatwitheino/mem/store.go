/*
 * Copyright 2026 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mem

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/msgops"
)

// SessionMeta provides summary info for the session list.
type SessionMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// Session holds the in-memory state for a single conversation.
type Session[M adk.MessageType] struct {
	ID        string
	CreatedAt time.Time

	filePath           string
	mu                 sync.Mutex
	messages           []M
	pendingInterruptID string // non-empty while the agent is paused awaiting human approval
	msgIdx             int    // A2UI component slot index at the point of last interrupt
}

// SetPendingInterruptID saves the interrupt ID so the approve endpoint can resume it.
func (s *Session[M]) SetPendingInterruptID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingInterruptID = id
}

// GetPendingInterruptID returns the stored interrupt ID, or "" if none is pending.
func (s *Session[M]) GetPendingInterruptID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingInterruptID
}

// SetMsgIdx stores the A2UI component slot counter so a resume can continue from it.
func (s *Session[M]) SetMsgIdx(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgIdx = idx
}

// GetMsgIdx returns the stored component slot counter.
func (s *Session[M]) GetMsgIdx() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.msgIdx
}

// Append adds a message to memory and persists it to disk.
func (s *Session[M]) Append(msg M) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg = msgops.NormalizeForSession(msg)
	s.messages = append(s.messages, msg)

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// GetMessages returns a snapshot of all messages.
func (s *Session[M]) GetMessages() []M {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]M, len(s.messages))
	copy(result, s.messages)
	return result
}

// Title derives a display title from the first user message.
func (s *Session[M]) Title() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, msg := range s.messages {
		if text := msgops.UserText(msg); text != "" {
			title := text
			if len([]rune(title)) > 60 {
				title = string([]rune(title)[:60]) + "..."
			}
			return title
		}
	}
	return "New Session"
}

// Store manages persisted sessions backed by JSONL files.
//
// File format:
//
//	{"type":"session","id":"...","created_at":"...","message_kind":"agentic"}   ← header (line 1)
//	{"role":"user","content_blocks":[...]}                                      ← message (lines 2+)
type Store[M adk.MessageType] struct {
	dir   string
	kind  msgops.Kind
	mu    sync.Mutex
	cache map[string]*Session[M]
}

// NewStore creates a new Store backed by the given directory (created if absent).
func NewStore[M adk.MessageType](dir string) (*Store[M], error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create session dir: %w", err)
	}
	return &Store[M]{
		dir:   dir,
		kind:  msgops.KindOf[M](),
		cache: make(map[string]*Session[M]),
	}, nil
}

// GetOrCreate returns the session for id, creating it if it does not exist.
func (s *Store[M]) GetOrCreate(id string) (*Session[M], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess, ok := s.cache[id]; ok {
		return sess, nil
	}

	filePath := filepath.Join(s.dir, id+".jsonl")

	var (
		sess *Session[M]
		err  error
	)
	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		sess, err = createSession[M](id, filePath)
	} else {
		sess, err = loadSession[M](filePath)
	}
	if err != nil {
		return nil, err
	}

	s.cache[id] = sess
	return sess, nil
}

// List returns metadata for all known sessions.
func (s *Store[M]) List() ([]SessionMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var metas []SessionMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".jsonl")

		if sess, ok := s.cache[id]; ok {
			metas = append(metas, SessionMeta{ID: id, Title: sess.Title(), CreatedAt: sess.CreatedAt})
			continue
		}

		sess, loadErr := loadSession[M](filepath.Join(s.dir, e.Name()))
		if loadErr != nil {
			continue
		}
		metas = append(metas, SessionMeta{ID: id, Title: sess.Title(), CreatedAt: sess.CreatedAt})
	}
	return metas, nil
}

// Delete removes the session file and evicts it from the cache.
func (s *Store[M]) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.dir, id+".jsonl")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(s.cache, id)
	return nil
}

// sessionHeader is the first JSONL line in every session file.
type sessionHeader struct {
	Type        string      `json:"type"`
	ID          string      `json:"id"`
	CreatedAt   time.Time   `json:"created_at"`
	MessageKind msgops.Kind `json:"message_kind,omitempty"`
}

func createSession[M adk.MessageType](id, filePath string) (*Session[M], error) {
	header := sessionHeader{
		Type:        "session",
		ID:          id,
		CreatedAt:   time.Now().UTC(),
		MessageKind: msgops.KindOf[M](),
	}
	data, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filePath, append(data, '\n'), 0o644); err != nil {
		return nil, err
	}
	return &Session[M]{
		ID:        id,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]M, 0),
	}, nil
}

func loadSession[M adk.MessageType](filePath string) (*Session[M], error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// First line: header
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty session file: %s", filePath)
	}
	var header sessionHeader
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, fmt.Errorf("bad session header in %s: %w", filePath, err)
	}
	if err := msgops.ValidateKind(header.MessageKind, msgops.KindOf[M](), true); err != nil {
		return nil, fmt.Errorf("cannot load session %s: %w", filePath, err)
	}

	sess := &Session[M]{
		ID:        header.ID,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]M, 0),
	}

	// Remaining lines: messages
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		msg, err := msgops.UnmarshalMessage[M]([]byte(line))
		if err != nil {
			continue // skip malformed lines
		}
		sess.messages = append(sess.messages, msgops.NormalizeForSession(msg))
	}

	return sess, scanner.Err()
}
