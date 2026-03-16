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
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	srcFlag := flag.String("src", "", "source dir: eino-ext repo root, skills dir, or installed skills root")
	destFlag := flag.String("dest", "", "destination dir (default: ./skills/eino-ext)")
	cleanFlag := flag.Bool("clean", false, "remove destination dir before syncing")
	flag.Parse()

	src := strings.TrimSpace(*srcFlag)
	if src == "" {
		src = strings.TrimSpace(os.Getenv("EINO_EXT_SKILLS_SRC"))
	}
	if src == "" {
		fmt.Fprintln(os.Stderr, "missing -src (or set EINO_EXT_SKILLS_SRC)")
		os.Exit(2)
	}

	dest := strings.TrimSpace(*destFlag)
	if dest == "" {
		dest = strings.TrimSpace(os.Getenv("EINO_EXT_SKILLS_DEST"))
	}
	if dest == "" {
		dest = filepath.Join(".", "skills", "eino-ext")
	}

	srcAbs, err := filepath.Abs(src)
	if err == nil {
		src = srcAbs
	}
	destAbs, err := filepath.Abs(dest)
	if err == nil {
		dest = destAbs
	}

	srcBase := resolveSourceBase(src)
	if srcBase == "" {
		fmt.Fprintf(os.Stderr, "invalid -src: %s\n", src)
		os.Exit(2)
	}

	if *cleanFlag {
		if err := os.RemoveAll(dest); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	want := []string{"eino-agent", "eino-component", "eino-compose", "eino-guide"}
	var copied []string
	for _, name := range want {
		srcDir := filepath.Join(srcBase, name)
		if !isDir(srcDir) {
			continue
		}
		destDir := filepath.Join(dest, name)
		if err := copyDir(srcDir, destDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := ensureSkillMD(destDir, name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		copied = append(copied, name)
	}

	if len(copied) == 0 {
		fmt.Fprintf(os.Stderr, "no skills found under %s (expected: %s)\n", srcBase, strings.Join(want, ", "))
		os.Exit(1)
	}

	sort.Strings(copied)
	fmt.Printf("Synced skills into %s: %s\n", dest, strings.Join(copied, ", "))
	fmt.Printf("Run with: EINO_EXT_SKILLS_DIR=%s go run ./cmd/ch09\n", dest)
}

func resolveSourceBase(src string) string {
	if isDir(filepath.Join(src, "skills")) {
		return filepath.Join(src, "skills")
	}
	if isDir(src) {
		return src
	}
	return ""
}

func ensureSkillMD(destDir, name string) error {
	skillPath := filepath.Join(destDir, "SKILL.md")
	if fileExists(skillPath) {
		return nil
	}

	entry := pickEntryFile(destDir)
	desc := defaultDescription(name)

	content := "---\n" +
		"name: " + name + "\n" +
		"description: " + desc + "\n" +
		"---\n\n" +
		"Use the documentation under this directory to answer questions about Eino.\n\n"
	if entry != "" {
		content += "Start with: " + entry + "\n"
	} else {
		content += "Start by listing markdown files in this directory.\n"
	}

	return os.WriteFile(skillPath, []byte(content), 0644)
}

func defaultDescription(name string) string {
	switch name {
	case "eino-guide":
		return "Entry point and navigation for Eino framework docs."
	case "eino-component":
		return "Component interfaces and implementations reference."
	case "eino-compose":
		return "Orchestration (Graph/Chain/Workflow) reference."
	case "eino-agent":
		return "ADK agents, middleware, runner reference."
	default:
		return "Eino skills documentation."
	}
}

func pickEntryFile(dir string) string {
	candidates := []string{
		"README.md",
		"readme.md",
		"index.md",
		"INDEX.md",
	}
	for _, c := range candidates {
		p := filepath.Join(dir, c)
		if fileExists(p) {
			return c
		}
	}

	var first string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".github" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), "SKILL.md") {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			rel, rerr := filepath.Rel(dir, path)
			if rerr == nil {
				first = filepath.ToSlash(rel)
				return errorsStopWalk{}
			}
		}
		return nil
	})
	return first
}

type errorsStopWalk struct{}

func (errorsStopWalk) Error() string { return "stop" }

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dest, rel), 0755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		destPath := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		return copyFile(path, destPath)
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
