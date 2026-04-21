package myagent

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryAndIdentityStoreReadWrite(t *testing.T) {
	workspace := t.TempDir()

	memoryStore := NewMemoryStore(workspace)
	identityStore := NewIdentityStore(workspace)

	if err := memoryStore.Write("prefers concise Chinese replies"); err != nil {
		t.Fatalf("memory write failed: %v", err)
	}
	if err := identityStore.Write("You are minichat."); err != nil {
		t.Fatalf("identity write failed: %v", err)
	}

	memoryContent, err := memoryStore.Read()
	if err != nil {
		t.Fatalf("memory read failed: %v", err)
	}
	if !strings.Contains(memoryContent, "prefers concise Chinese replies") {
		t.Fatalf("unexpected memory content: %q", memoryContent)
	}

	identityContent, err := identityStore.Read()
	if err != nil {
		t.Fatalf("identity read failed: %v", err)
	}
	if !strings.Contains(identityContent, "You are minichat.") {
		t.Fatalf("unexpected identity content: %q", identityContent)
	}
}

func TestIsProtectedContextPath(t *testing.T) {
	workspace := t.TempDir()

	if !isProtectedContextPath(workspace, filepath.Join(workspace, "memory", memoryFileName)) {
		t.Fatal("expected memory path to be protected")
	}
	if !isProtectedContextPath(workspace, filepath.Join(workspace, identityFileName)) {
		t.Fatal("expected identity path to be protected")
	}
	if isProtectedContextPath(workspace, filepath.Join(workspace, "notes.md")) {
		t.Fatal("expected notes path to be unprotected")
	}
}

func TestEnsureWorkspaceRuntimeCreatesDefaultMemoryTemplate(t *testing.T) {
	workspace := t.TempDir()

	ws, err := ensureWorkspaceRuntime(workspace)
	if err != nil {
		t.Fatalf("ensure workspace runtime failed: %v", err)
	}

	content, err := readOptionalFile(filepath.Join(ws.memoryDir, memoryFileName))
	if err != nil {
		t.Fatalf("read memory file failed: %v", err)
	}
	if !strings.Contains(content, "# Long-term Memory") {
		t.Fatalf("expected memory template header, got: %q", content)
	}
	if !strings.Contains(content, "## User Information") {
		t.Fatalf("expected user information section, got: %q", content)
	}
	if !strings.Contains(content, "## Preferences") {
		t.Fatalf("expected preferences section, got: %q", content)
	}
}
