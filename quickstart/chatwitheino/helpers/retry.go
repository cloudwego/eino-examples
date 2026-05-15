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
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"

	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/msgops"
)

// ApplyMessageModelRetry enables model-call retries for classic schema.Message
// runs only. AgenticMessage replays provider-native Responses items, so automatic
// full model-step retries stay disabled until AgenticModel has explicit retry
// semantics for partially generated response items and tool state.
func ApplyMessageModelRetry[M adk.MessageType](cfg *deep.TypedConfig[M]) {
	if msgops.KindOf[M]() != msgops.KindMessage {
		return
	}

	cfg.ModelRetryConfig = &adk.TypedModelRetryConfig[M]{
		MaxRetries: 5,
		IsRetryAble: func(_ context.Context, err error) bool {
			return strings.Contains(err.Error(), "429") ||
				strings.Contains(err.Error(), "Too Many Requests") ||
				strings.Contains(err.Error(), "qpm limit")
		},
	}
}
