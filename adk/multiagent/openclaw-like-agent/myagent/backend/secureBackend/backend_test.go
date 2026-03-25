package secureBackend

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk/filesystem"
)

func TestAllowPathsPermitReadButNotWriteOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	externalSkillsDir := filepath.Join(t.TempDir(), "skills")
	skillFile := filepath.Join(externalSkillsDir, "demo", "SKILL.md")

	if err := os.MkdirAll(filepath.Dir(skillFile), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(skillFile, []byte("# demo\n"), 0o644); err != nil {
		t.Fatalf("write skill file failed: %v", err)
	}

	sep := regexp.QuoteMeta(string(filepath.Separator))
	allowPattern := regexp.MustCompile("^" + regexp.QuoteMeta(filepath.Clean(externalSkillsDir)) + "(?:$|" + sep + ")")

	backend, err := New(&Config{
		Workspace:  workspace,
		Restrict:   true,
		AllowPaths: []*regexp.Regexp{allowPattern},
	})
	if err != nil {
		t.Fatalf("new backend failed: %v", err)
	}

	content, err := backend.Read(context.Background(), &filesystem.ReadRequest{FilePath: skillFile})
	if err != nil {
		t.Fatalf("read allowlisted skill failed: %v", err)
	}
	if !strings.Contains(content.Content, "# demo") {
		t.Fatalf("unexpected read content: %q", content.Content)
	}

	err = backend.Write(context.Background(), &filesystem.WriteRequest{
		FilePath: skillFile,
		Content:  "mutated",
	})
	if err == nil {
		t.Fatal("expected write outside workspace to be denied")
	}
	if !strings.Contains(err.Error(), "path escapes workspace") {
		t.Fatalf("unexpected write error: %v", err)
	}
}
