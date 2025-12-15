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
	"bytes"
	"context"
	"encoding/gob"

	"github.com/cloudwego/eino/schema"
)

// MemoryStore persists and restores short-term conversation history.
// Implementations are responsible for storing a slice of messages under a session key.
type MemoryStore interface {
	Write(ctx context.Context, sessionID string, msgs []*schema.Message) error
	Read(ctx context.Context, sessionID string) ([]*schema.Message, error)
	Query(ctx context.Context, sessionID string, text string, limit int) ([]*schema.Message, error)
}

// Gob registrations for eino message types are provided by the framework; no manual registration needed here.

// EncodeMessages serializes messages using Gob.
func EncodeMessages(msgs []*schema.Message) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(msgs); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeMessages deserializes messages previously encoded by EncodeMessages.
func DecodeMessages(b []byte) ([]*schema.Message, error) {
	if len(b) == 0 {
		return nil, nil
	}
	dec := gob.NewDecoder(bytes.NewReader(b))
	var msgs []*schema.Message
	if err := dec.Decode(&msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}
