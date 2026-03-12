/*
 * Copyright 2025 CloudWeGo Authors
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

package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// InMemoryStore keeps serialized messages in a process-local map.
// Suitable for demos/tests; not shared across processes.
type InMemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[string][]byte)}
}

// Write encodes and stores messages for the given key.
func (s *InMemoryStore) Write(ctx context.Context, sessionID string, msgs []*schema.Message) error {
	b, err := EncodeMessages(msgs)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.data[sessionID] = b
	s.mu.Unlock()
	return nil
}

// Read returns decoded messages for the given session; returns nil if absent.
func (s *InMemoryStore) Read(ctx context.Context, sessionID string) ([]*schema.Message, error) {
	s.mu.RLock()
	b := s.data[sessionID]
	s.mu.RUnlock()
	return DecodeMessages(b)
}

// Query performs a simple substring search on message contents for the session.
func (s *InMemoryStore) Query(ctx context.Context, sessionID string, text string, limit int) ([]*schema.Message, error) {
	msgs, err := s.Read(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 || text == "" {
		return nil, nil
	}
	q := strings.ToLower(text)
	out := make([]*schema.Message, 0, limit)
	for _, m := range msgs {
		if m == nil {
			continue
		}
		if strings.Contains(strings.ToLower(m.Content), q) {
			out = append(out, m)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
