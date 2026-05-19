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

package msgops

import (
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// Kind identifies the message representation used by a chatwitheino run.
type Kind string

const (
	KindMessage Kind = "message"
	KindAgentic Kind = "agentic"
)

// KindFromEnv reads MESSAGE_KIND. AgenticMessage is the default representation;
// set MESSAGE_KIND=message only when explicitly exercising the legacy path.
func KindFromEnv() Kind {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MESSAGE_KIND"))) {
	case string(KindAgentic), "agenticmessage", "agentic_message":
		return KindAgentic
	case string(KindMessage):
		return KindMessage
	default:
		return KindAgentic
	}
}

// KindOf returns the message kind represented by M.
func KindOf[M adk.MessageType]() Kind {
	var zero M
	switch any(zero).(type) {
	case *schema.AgenticMessage:
		return KindAgentic
	default:
		return KindMessage
	}
}

// DefaultSessionDir returns the default session directory for the current kind.
func DefaultSessionDir(kind Kind) string {
	if kind == KindAgentic {
		if dir := strings.TrimSpace(os.Getenv("SESSION_DIR_AGENTIC")); dir != "" {
			return dir
		}
		return "./data/sessions_agentic"
	}
	if dir := strings.TrimSpace(os.Getenv("SESSION_DIR")); dir != "" {
		return dir
	}
	return "./data/sessions"
}

// ValidateKind rejects files written for a different message representation.
func ValidateKind(stored, target Kind, legacyMessageOK bool) error {
	if stored == "" && target == KindMessage && legacyMessageOK {
		return nil
	}
	if stored == "" {
		return fmt.Errorf("session file has no message_kind; current MESSAGE_KIND=%s", target)
	}
	if stored != target {
		return fmt.Errorf("session file uses message_kind=%s; current MESSAGE_KIND=%s", stored, target)
	}
	return nil
}
