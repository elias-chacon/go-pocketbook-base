package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sample = "// examples/sample.go\npackage examples\n\n// Hello says hi.\nfunc Hello() string { return \"hi\" }\n\nfunc other() {}\n"

func TestSyncFillsAndRefreshesBlocks(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "examples", "sample.go"), sample)
	page := filepath.Join(root, "docs", "en", "page.md")
	write(t, page, "Intro\n\n```go\n// examples/sample.go\n```\n\n```go\n// examples/sample.go#Hello\nstale text\n```\n\n```sh\nnot touched\n```\n")

	stale, err := syncAll(root, false)
	if err != nil || len(stale) != 1 {
		t.Fatalf("check: stale=%v err=%v", stale, err)
	}
	if data, _ := os.ReadFile(page); strings.Contains(string(data), "func other") {
		t.Fatal("check mode must not write")
	}

	if _, err := syncAll(root, true); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(page)
	got := string(data)
	for _, want := range []string{
		"```go\n// examples/sample.go\npackage examples\n", // the whole file, header once
		"func other() {}",
		"```go\n// examples/sample.go#Hello\n// Hello says hi.\nfunc Hello() string { return \"hi\" }\n```", // only the function, with its doc
		"```sh\nnot touched\n```",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("page lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "stale text") || strings.Count(got, "// examples/sample.go\n") != 1 {
		t.Errorf("old content must be replaced:\n%s", got)
	}

	if stale, _ := syncAll(root, false); len(stale) != 0 {
		t.Errorf("after a sync nothing is stale: %v", stale)
	}
}

func TestSyncReportsUnknownSources(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "examples", "sample.go"), sample)
	write(t, filepath.Join(root, "docs", "a.md"), "```go\n// examples/missing.go\n```\n")
	if _, err := syncAll(root, false); err == nil {
		t.Error("a missing file must be an error")
	}
	write(t, filepath.Join(root, "docs", "a.md"), "```go\n// examples/sample.go#Nope\n```\n")
	if _, err := syncAll(root, false); err == nil {
		t.Error("a missing function must be an error")
	}
}

func TestSyncIsLineEndingAgnostic(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "examples", "sample.go"), strings.ReplaceAll(sample, "\n", "\r\n"))
	write(t, filepath.Join(root, "docs", "a.md"), "```go\r\n// examples/sample.go#Hello\r\n```\r\n")
	if _, err := syncAll(root, true); err != nil {
		t.Fatal(err)
	}
	if stale, _ := syncAll(root, false); len(stale) != 0 {
		t.Errorf("CRLF sources must not leave pages stale: %v", stale)
	}
}

func TestNoDocsDirectoryIsFine(t *testing.T) {
	if stale, err := syncAll(t.TempDir(), false); err != nil || stale != nil {
		t.Errorf("stale=%v err=%v", stale, err)
	}
}

// The documentation of this repository must always match the tested examples.
func TestRepositoryDocsAreInSync(t *testing.T) {
	stale, err := syncAll(filepath.Join("..", ".."), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) > 0 {
		t.Errorf("out of date (run `go run ./tools/docsync`): %v", stale)
	}
}
