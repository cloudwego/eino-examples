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

package main

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

// generateReport simulates a long-running background report generation job.
// It sends the final URL or an error via the notify channel.
func generateReport(ctx context.Context, content string, notify chan<- reportResult) {
	select {
	case <-ctx.Done():
		notify <- reportResult{"", ctx.Err()}
		close(notify)
		return
	case <-time.After(2 * time.Second):
		if strings.Contains(strings.ToLower(content), "error") {
			notify <- reportResult{"", errors.New("report generation failed")}
			close(notify)
			return
		}
		url := "https://example.com/report/" + strings.ReplaceAll(strings.ToLower(content), " ", "-")
		notify <- reportResult{url, nil}
		close(notify)
		return
	}
}

// transcribeLive simulates a live transcription service that emits tokens over time.
// It may emit an error mid-stream for demonstration when encountering the word "error".
func transcribeLive(ctx context.Context, phrase string) *schema.StreamReader[string] {
	sr, sw := schema.Pipe[string](utf8.RuneCountInString(phrase))

	go func() {
		defer sw.Close()

		splitter := func(r rune) bool { return r == ' ' || r == '-' || r == '/' }
		for _, w := range strings.FieldsFunc(phrase, splitter) {
			select {
			case <-ctx.Done():
				sw.Send("", ctx.Err())
				return
			default:
			}

			if strings.EqualFold(w, "error") {
				sw.Send("", errors.New("transcription stream error"))
				return
			}

			sw.Send(w, nil)
			time.Sleep(300 * time.Millisecond)
		}
	}()

	return sr
}

type reportResult struct {
	url string
	err error
}
