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

package batch

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// batchBridgeStore implements compose.CheckPointStore for batch processing.
// It stores checkpoint data keyed by batch index, allowing each sub-task
// to have its own checkpoint namespace.
//
// This store is used internally by BatchNode and is not meant for external use.
// For interrupt/resume, the BatchNode stores its state via CompositeInterrupt,
// not through this checkpoint store.
type batchBridgeStore struct {
	mu   sync.RWMutex
	data map[int][]byte // index -> checkpoint data
}

// newBatchBridgeStore creates a new empty checkpoint store.
func newBatchBridgeStore() *batchBridgeStore {
	return &batchBridgeStore{
		data: make(map[int][]byte),
	}
}

// makeBatchCheckpointID creates a checkpoint ID for a given batch index.
// Format: "batch_0", "batch_1", etc.
func makeBatchCheckpointID(index int) string {
	return fmt.Sprintf("batch_%d", index)
}

// parseBatchIndex extracts the batch index from a checkpoint ID.
// Returns error if the ID format is invalid.
func parseBatchIndex(checkPointID string) (int, error) {
	if !strings.HasPrefix(checkPointID, "batch_") {
		return 0, fmt.Errorf("invalid batch checkpoint ID: %s", checkPointID)
	}
	indexStr := strings.TrimPrefix(checkPointID, "batch_")
	return strconv.Atoi(indexStr)
}

// Get retrieves checkpoint data for a batch index.
// Implements compose.CheckPointStore interface.
func (m *batchBridgeStore) Get(_ context.Context, checkPointID string) ([]byte, bool, error) {
	index, err := parseBatchIndex(checkPointID)
	if err != nil {
		return nil, false, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.data[index]
	return data, ok, nil
}

// Set stores checkpoint data for a batch index.
// Implements compose.CheckPointStore interface.
func (m *batchBridgeStore) Set(_ context.Context, checkPointID string, checkPoint []byte) error {
	index, err := parseBatchIndex(checkPointID)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[index] = checkPoint
	return nil
}
