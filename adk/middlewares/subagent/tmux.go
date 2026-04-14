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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// tmuxMode represents the available tmux integration level.
type tmuxMode int

const (
	tmuxModeNone     tmuxMode = iota // tmux not available
	tmuxModeExternal                 // tmux available but not inside a session
	tmuxModeInside                   // running inside a tmux session
)

// tmuxManager manages tmux windows for subagent tasks.
type tmuxManager struct {
	mode        tmuxMode
	sessionName string // current tmux session name
	tmpDir      string // temp directory for log files

	mu      sync.Mutex
	windows map[string]*tmuxWindow
}

// tmuxWindow tracks a single tmux window for a task.
type tmuxWindow struct {
	taskID     string
	windowName string
	logPath    string   // temp file path for event output
	writer     *os.File // append-mode file handle
	writerMu   sync.Mutex
	closed     bool
}

// detectTmuxMode checks whether tmux is available and whether we're inside a session.
func detectTmuxMode() tmuxMode {
	if _, err := exec.LookPath("tmux"); err != nil {
		return tmuxModeNone
	}
	if os.Getenv("TMUX") != "" {
		return tmuxModeInside
	}
	return tmuxModeExternal
}

// newTmuxManager detects tmux availability and creates a manager.
func newTmuxManager() *tmuxManager {
	mode := detectTmuxMode()
	tm := &tmuxManager{
		mode:    mode,
		windows: make(map[string]*tmuxWindow),
	}

	if mode == tmuxModeNone {
		return tm
	}

	tmpDir, err := os.MkdirTemp("", "eino-tmux-*")
	if err != nil {
		log.Printf("Warning: failed to create temp dir: %v", err)
		tm.mode = tmuxModeNone
		return tm
	}
	tm.tmpDir = tmpDir

	if mode == tmuxModeInside {
		out, err := exec.Command("tmux", "display-message", "-p", "#S").Output()
		if err != nil {
			log.Printf("Warning: failed to get tmux session name: %v", err)
			tm.mode = tmuxModeNone
			return tm
		}
		tm.sessionName = strings.TrimSpace(string(out))
	} else {
		tm.sessionName = "eino-subagents"
		_ = exec.Command("tmux", "new-session", "-d", "-s", tm.sessionName).Run()
	}

	return tm
}

// CreateWindow creates a new tmux window for the given task.
// Uses a regular temp file + "tail -f" with "stty -echo" to prevent
// arrow key escape sequences from being echoed.
func (tm *tmuxManager) CreateWindow(taskID, description string) (string, error) {
	if tm.mode == tmuxModeNone {
		return "", nil
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	windowName := taskID
	logPath := filepath.Join(tm.tmpDir, windowName+".log")

	// Create the log file with a header
	f, err := os.Create(logPath)
	if err != nil {
		return "", fmt.Errorf("create log file: %w", err)
	}
	fmt.Fprintf(f, "=== SubAgent Task [%s]: %s ===\n", taskID, description)
	fmt.Fprintf(f, "=== Started: %s ===\n\n", time.Now().Format("15:04:05"))

	// tmux window: disable echo so arrow keys don't produce garbage, then tail -f
	shellCmd := fmt.Sprintf(
		"stty -echo 2>/dev/null; tail -n +1 -f '%s'",
		logPath,
	)

	target := tm.sessionName + ":"
	cmd := exec.Command("tmux", "new-window", "-d", "-t", target, "-n", windowName, "bash", "-c", shellCmd)
	if err := cmd.Run(); err != nil {
		f.Close()
		os.Remove(logPath)
		return "", fmt.Errorf("tmux new-window: %w", err)
	}

	w := &tmuxWindow{
		taskID:     taskID,
		windowName: windowName,
		logPath:    logPath,
		writer:     f,
	}
	tm.windows[taskID] = w

	return windowName, nil
}

// WriteEvent writes a formatted event line to the task's log file.
// The tmux window's "tail -f" picks it up automatically.
func (tm *tmuxManager) WriteEvent(taskID string, entry logEntry) {
	if tm.mode == tmuxModeNone {
		return
	}

	tm.mu.Lock()
	w, ok := tm.windows[taskID]
	tm.mu.Unlock()

	if !ok {
		return
	}

	w.writerMu.Lock()
	defer w.writerMu.Unlock()

	if w.writer == nil || w.closed {
		return
	}

	lines := entry.renderFull(120)
	for _, line := range lines {
		fmt.Fprintln(w.writer, line)
	}
	w.writer.Sync()
}

// MarkComplete writes a completion banner and closes the log file.
func (tm *tmuxManager) MarkComplete(taskID string, status string, result string) {
	if tm.mode == tmuxModeNone {
		return
	}

	tm.mu.Lock()
	w, ok := tm.windows[taskID]
	tm.mu.Unlock()

	if !ok {
		return
	}

	w.writerMu.Lock()
	defer w.writerMu.Unlock()

	if w.closed {
		return
	}
	w.closed = true

	if w.writer != nil {
		fmt.Fprintf(w.writer, "\n%s\n", strings.Repeat("=", 60))
		fmt.Fprintf(w.writer, "Task [%s] %s at %s\n", taskID, status, time.Now().Format("15:04:05"))
		if result != "" {
			fmt.Fprintf(w.writer, "\nResult:\n%s\n", result)
		}
		fmt.Fprintf(w.writer, "%s\n", strings.Repeat("=", 60))
		w.writer.Sync()
		w.writer.Close()
	}

	// Rename tmux window to indicate completion
	newName := taskID + "_" + status
	_ = exec.Command("tmux", "rename-window", "-t",
		fmt.Sprintf("%s:%s", tm.sessionName, w.windowName), newName).Run()
	w.windowName = newName
}

// Mode returns the current tmux integration level.
func (tm *tmuxManager) Mode() tmuxMode {
	return tm.mode
}

// SessionName returns the session name (for display hints).
func (tm *tmuxManager) SessionName() string {
	return tm.sessionName
}

// Cleanup removes all log files and the temp directory.
func (tm *tmuxManager) Cleanup() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, w := range tm.windows {
		w.writerMu.Lock()
		if w.writer != nil && !w.closed {
			w.writer.Close()
		}
		w.writerMu.Unlock()
	}

	if tm.tmpDir != "" {
		os.RemoveAll(tm.tmpDir)
	}
}
