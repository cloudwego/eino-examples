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

package helpers

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/schema"
)

func TestApplyMessageModelRetry(t *testing.T) {
	cfg := &deep.TypedConfig[*schema.Message]{}
	ApplyMessageModelRetry(cfg)

	if cfg.ModelRetryConfig.MaxRetries != 5 {
		t.Fatalf("MaxRetries = %d, want 5", cfg.ModelRetryConfig.MaxRetries)
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "rate limit code", err: errors.New("429"), want: true},
		{name: "rate limit message", err: errors.New("Too Many Requests"), want: true},
		{name: "QPM limit", err: errors.New("qpm limit exceeded"), want: true},
		{name: "unrelated error", err: errors.New("connection refused")},
		{name: "no error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := cfg.ModelRetryConfig.ShouldRetry(context.Background(), &adk.TypedRetryContext[*schema.Message]{Err: tt.err})
			got := decision != nil && decision.Retry
			if got != tt.want {
				t.Errorf("ShouldRetry() = %t, want %t", got, tt.want)
			}
		})
	}
}
