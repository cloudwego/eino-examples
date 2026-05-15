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
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
)

// ApplyMessageModelRetry enables model-call retries for transient rate-limit
// errors.
func ApplyMessageModelRetry[M adk.MessageType](cfg *deep.TypedConfig[M]) {
	cfg.ModelRetryConfig = &adk.TypedModelRetryConfig[M]{
		MaxRetries: 5,
		IsRetryAble: func(_ context.Context, err error) bool {
			return strings.Contains(err.Error(), "429") ||
				strings.Contains(err.Error(), "Too Many Requests") ||
				strings.Contains(err.Error(), "qpm limit")
		},
	}
}

func IsModelRetryInProgress(err error) bool {
	var willRetry *adk.WillRetryError
	return errors.As(err, &willRetry)
}

func LogModelRetry(w io.Writer, err error) bool {
	if !IsModelRetryInProgress(err) {
		return false
	}
	if w != nil {
		fmt.Fprintf(w, "\n[retry] model call failed, retrying: %v\n", err)
	}
	return true
}
