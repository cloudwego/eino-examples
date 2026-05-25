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

// Package a2ui implements a subset of the A2UI v0.8 specification for streaming
// agent output to a web UI over JSONL/SSE.
package a2ui

import "encoding/json"

// Message is the top-level A2UI envelope. Exactly one field is populated per message.
type Message struct {
	BeginRendering   *BeginRenderingMsg   `json:"beginRendering,omitempty"`
	SurfaceUpdate    *SurfaceUpdateMsg    `json:"surfaceUpdate,omitempty"`
	DataModelUpdate  *DataModelUpdateMsg  `json:"dataModelUpdate,omitempty"`
	DeleteSurface    *DeleteSurfaceMsg    `json:"deleteSurface,omitempty"`
	InterruptRequest *InterruptRequestMsg `json:"interruptRequest,omitempty"`
}

// BeginRenderingMsg signals the start of a rendering session for a surface.
type BeginRenderingMsg struct {
	SurfaceID string `json:"surfaceId"`
	Root      string `json:"root"`
}

// SurfaceUpdateMsg adds or updates components on a surface.
type SurfaceUpdateMsg struct {
	SurfaceID  string      `json:"surfaceId"`
	Components []Component `json:"components"`
}

// DataModelUpdateMsg updates the data bindings used by Text components.
type DataModelUpdateMsg struct {
	SurfaceID string        `json:"surfaceId"`
	Contents  []DataContent `json:"contents"`
}

// DeleteSurfaceMsg removes a surface from the renderer.
type DeleteSurfaceMsg struct {
	SurfaceID string `json:"surfaceId"`
}

// Component is a named UI component definition.
type Component struct {
	ID        string         `json:"id"`
	Component ComponentValue `json:"component"`
}

// ComponentValue holds exactly one component type.
type ComponentValue struct {
	Text   *TextComp   `json:"Text,omitempty"`
	Column *ColumnComp `json:"Column,omitempty"`
	Card   *CardComp   `json:"Card,omitempty"`
	Row    *RowComp    `json:"Row,omitempty"`
}

// TextComp renders text. If DataKey is set, the value is read from the data model.
type TextComp struct {
	Value     string `json:"value,omitempty"`
	DataKey   string `json:"dataKey,omitempty"`
	UsageHint string `json:"usageHint,omitempty"` // "caption" | "body" | "title"
}

// ColumnComp lays out its children vertically.
type ColumnComp struct {
	Children []string `json:"children"`
}

// CardComp wraps its children in a card container.
type CardComp struct {
	Children []string `json:"children"`
}

// RowComp lays out its children horizontally.
type RowComp struct {
	Children []string `json:"children"`
}

// DataContent is a key-value binding for use in DataModelUpdateMsg.
type DataContent struct {
	Key         string `json:"key"`
	ValueString string `json:"valueString,omitempty"`
}

// InterruptRequestMsg is sent when the agent is interrupted and awaits human approval.
// The receiver should display the description to the user and call back with the approval decision.
type InterruptRequestMsg struct {
	InterruptID string `json:"interruptId"`
	Description string `json:"description"` // human-readable reason (from the interrupt's Info)
}

// Encode serializes a Message to JSON followed by a newline byte.
func Encode(msg Message) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
