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

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/adk"
	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type responseInspector func(*schema.AgenticMessage) (status, reason string)

func newMaxOutputTokenRetryConfig(retryMaxTokens []int, inspect responseInspector) *adk.TypedModelRetryConfig[*schema.AgenticMessage] {
	return &adk.TypedModelRetryConfig[*schema.AgenticMessage]{
		MaxRetries: len(retryMaxTokens),
		ShouldRetry: func(_ context.Context, retryCtx *adk.TypedRetryContext[*schema.AgenticMessage]) *adk.TypedRetryDecision[*schema.AgenticMessage] {
			if retryCtx.Err != nil || !isMaxOutputTokensIncomplete(retryCtx.OutputMessage, inspect) {
				return nil
			}

			status, reason := inspect(retryCtx.OutputMessage)
			idx := retryCtx.RetryAttempt - 1
			if idx >= len(retryMaxTokens) {
				return &adk.TypedRetryDecision[*schema.AgenticMessage]{
					RewriteError: fmt.Errorf("response remained incomplete after %d retries: status=%q reason=%q", len(retryMaxTokens), status, reason),
				}
			}

			nextMaxTokens := retryMaxTokens[idx]
			fmt.Printf("[retry] attempt=%d rejected status=%q reason=%q -> max_output_tokens=%d\n",
				retryCtx.RetryAttempt, status, reason, nextMaxTokens)

			return &adk.TypedRetryDecision[*schema.AgenticMessage]{
				Retry:             true,
				AdditionalOptions: []einoModel.Option{einoModel.WithMaxTokens(nextMaxTokens)},
				Backoff:           100 * time.Millisecond,
				RejectReason:      fmt.Sprintf("response incomplete because %s; retrying with max_output_tokens=%d", reason, nextMaxTokens),
			}
		},
	}
}

func openAIStatusAndReason(msg *schema.AgenticMessage) (string, string) {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.OpenAIExtension == nil {
		return "", ""
	}
	ext := msg.ResponseMeta.OpenAIExtension
	if ext.IncompleteDetails == nil {
		return string(ext.Status), ""
	}
	return string(ext.Status), ext.IncompleteDetails.Reason
}

func arkStatusAndReason(msg *schema.AgenticMessage) (string, string) {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Extension == nil {
		return "", ""
	}

	switch ext := msg.ResponseMeta.Extension.(type) {
	case *agenticark.ResponseMetaExtension:
		if ext.IncompleteDetails == nil {
			return string(ext.Status), ""
		}
		return string(ext.Status), ext.IncompleteDetails.Reason
	case agenticark.ResponseMetaExtension:
		if ext.IncompleteDetails == nil {
			return string(ext.Status), ""
		}
		return string(ext.Status), ext.IncompleteDetails.Reason
	default:
		return "", ""
	}
}

func isMaxOutputTokensIncomplete(msg *schema.AgenticMessage, inspect responseInspector) bool {
	status, reason := inspect(msg)
	return strings.EqualFold(strings.TrimSpace(status), "incomplete") && isTokenLimitReason(reason)
}

func isTokenLimitReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case tokenLimitReason, "max_tokens", "length":
		return true
	default:
		return false
	}
}
