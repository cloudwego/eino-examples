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

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
)

// RedisStore persists serialized messages in Redis under the provided session key.
type RedisStore struct {
	cli *redis.Client
}

func NewRedisStore(cli *redis.Client) *RedisStore {
	return &RedisStore{cli: cli}
}

// Write encodes and stores messages using Redis SET.
func (s *RedisStore) Write(ctx context.Context, sessionID string, msgs []*schema.Message) error {
	b, err := EncodeMessages(msgs)
	if err != nil {
		return err
	}
	return s.cli.Set(ctx, sessionID, b, 0).Err()
}

// Read returns decoded messages from Redis GET; returns nil if not found.
func (s *RedisStore) Read(ctx context.Context, sessionID string) ([]*schema.Message, error) {
	res, err := s.cli.Get(ctx, sessionID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return DecodeMessages(res)
}

func (s *RedisStore) Query(ctx context.Context, sessionID string, text string, limit int) ([]*schema.Message, error) {
	msgs, err := s.Read(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 || text == "" {
		return nil, nil
	}
	out := make([]*schema.Message, 0, limit)
	q := strings.ToLower(text)
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

// NewMiniRedisClient starts an embedded Redis server for local demos/tests.
func NewMiniRedisClient() (*redis.Client, func(), error) {
	srv, err := miniredis.Run()
	if err != nil {
		return nil, nil, err
	}
	cli := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	closer := func() { srv.Close() }
	return cli, closer, nil
}
