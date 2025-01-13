package todo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var defaultStorage *Storage

type Storage struct {
	filePath string
	mu       sync.RWMutex
	cache    map[string]*Todo
	dirty    bool
}

func GetDefaultStorage() *Storage {
	if defaultStorage == nil {
		InitDefaultStorage("./data/todo")
	}
	return defaultStorage
}

func InitDefaultStorage(dataDir string) error {
	s, err := NewStorage(dataDir)
	if err != nil {
		return err
	}
	defaultStorage = s
	return nil
}

func NewStorage(dataDir string) (*Storage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}
	s := &Storage{
		filePath: filepath.Join(dataDir, "todos.jsonl"),
		cache:    make(map[string]*Todo),
	}

	if err := s.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load from disk: %v", err)
	}

	return s, nil
}

func (s *Storage) loadFromDisk() error {
	file, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var todo Todo
		if err := json.Unmarshal(scanner.Bytes(), &todo); err != nil {
			return fmt.Errorf("failed to unmarshal todo: %v", err)
		}
		s.cache[todo.ID] = &todo
	}

	return scanner.Err()
}

func (s *Storage) Add(todo *Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo.CreatedAt = time.Now().Format(time.RFC3339)
	todo.IsDeleted = false
	s.cache[todo.ID] = todo

	// 直接追加到文件末尾
	file, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	data, err := json.Marshal(todo)
	if err != nil {
		return fmt.Errorf("failed to marshal todo: %v", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write todo: %v", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %v", err)
	}

	return nil
}

func (s *Storage) List(params *ListParams) ([]*Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var activeTodos, completedTodos []*Todo
	for _, todo := range s.cache {
		if todo.IsDeleted {
			continue
		}

		if params.Query != "" && !contains(todo.Title, params.Query) && !contains(todo.Content, params.Query) {
			continue
		}

		if params.IsDone != nil {
			if todo.Completed != *params.IsDone {
				continue
			}
		}

		if todo.Completed {
			completedTodos = append(completedTodos, todo)
		} else {
			activeTodos = append(activeTodos, todo)
		}
	}

	// 按创建时间排序（最新的在前面）
	sort.Slice(activeTodos, func(i, j int) bool {
		return activeTodos[i].CreatedAt > activeTodos[j].CreatedAt
	})
	sort.Slice(completedTodos, func(i, j int) bool {
		return completedTodos[i].CreatedAt > completedTodos[j].CreatedAt
	})

	// 合并列表：未完成的在前，已完成的在后
	todos := append(activeTodos, completedTodos...)

	if params.Limit != nil && len(todos) > *params.Limit {
		todos = todos[:*params.Limit]
	}

	return todos, nil
}

func (s *Storage) Update(todo *Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.cache[todo.ID]
	if !exists || existing.IsDeleted {
		return fmt.Errorf("todo not found: %s", todo.ID)
	}

	// 只更新非空字段
	updated := *existing // 创建副本
	if todo.Title != "" {
		updated.Title = todo.Title
	}
	if todo.Content != "" {
		updated.Content = todo.Content
	}
	if todo.Deadline != "" {
		updated.Deadline = todo.Deadline
	}
	// Completed 字段需要特殊处理，因为它是布尔值
	if todo.Completed != existing.Completed {
		updated.Completed = todo.Completed
	}

	s.cache[todo.ID] = &updated
	s.dirty = true

	return s.syncToDisk()
}

func (s *Storage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, exists := s.cache[id]
	if !exists || todo.IsDeleted {
		return fmt.Errorf("todo not found: %s", id)
	}

	// 标记删除
	todo.IsDeleted = true
	s.dirty = true

	return s.syncToDisk()
}

func (s *Storage) syncToDisk() error {
	if !s.dirty {
		return nil
	}

	// 创建临时文件
	tmpFile := s.filePath + ".tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer file.Close()

	// 写入数据到临时文件
	for _, todo := range s.cache {
		data, err := json.Marshal(todo)
		if err != nil {
			os.Remove(tmpFile) // 清理临时文件
			return fmt.Errorf("failed to marshal todo: %v", err)
		}

		if _, err := file.Write(append(data, '\n')); err != nil {
			os.Remove(tmpFile) // 清理临时文件
			return fmt.Errorf("failed to write todo: %v", err)
		}
	}

	// 确保所有数据都写入磁盘
	if err := file.Sync(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to sync file: %v", err)
	}

	// 关闭文件
	if err := file.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to close file: %v", err)
	}

	// 备份现有文件（如果存在）
	if _, err := os.Stat(s.filePath); err == nil {
		backupFile := s.filePath + ".bak"
		if err := os.Rename(s.filePath, backupFile); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("failed to backup file: %v", err)
		}
	}

	// 将临时文件重命名为正式文件
	if err := os.Rename(tmpFile, s.filePath); err != nil {
		// 如果重命名失败，尝试恢复备份
		if backupErr := os.Rename(s.filePath+".bak", s.filePath); backupErr != nil {
			return fmt.Errorf("failed to rename temp file and restore backup: %v, backup error: %v", err, backupErr)
		}
		return fmt.Errorf("failed to rename temp file: %v", err)
	}

	// 删除备份文件
	os.Remove(s.filePath + ".bak")

	s.dirty = false
	return nil
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
