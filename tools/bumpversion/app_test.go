package main

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "apps", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "apps", "x", ".version"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func fixed(s string) func(string, string) (string, error) {
	return func(string, string) (string, error) { return s, nil }
}

func TestAnAppKeepsItsVersionUntilItsSourcesChange(t *testing.T) {
	root := fixture(t, "1.2.3\n")
	v, changed, err := bumpApp(root, "x", "auto", fixed("aaa"))
	if err != nil || !changed || v.String() != "1.2.4" {
		t.Fatalf("a version never stamped counts as changed: %v %v %v", v, changed, err)
	}
	v, changed, _ = bumpApp(root, "x", "auto", fixed("aaa"))
	if changed || v.String() != "1.2.4" {
		t.Errorf("the same sources must keep the version, got %v changed=%v", v, changed)
	}
	if _, changed, _ = bumpApp(root, "x", "check", fixed("bbb")); !changed {
		t.Error("check must see the change")
	}
	if v, _, _ = bumpApp(root, "x", "check", fixed("bbb")); v.String() != "1.2.4" {
		t.Errorf("check must not bump, got %v", v)
	}
	v, _, _ = bumpApp(root, "x", "auto", fixed("bbb"))
	if v.String() != "1.2.5" {
		t.Errorf("changed sources bump BUILD, got %v", v)
	}
	v, _, _ = bumpApp(root, "x", "minor", fixed("bbb"))
	if v.String() != "1.3.0" || recorded(filepath.Join(root, "apps", "x", ".version")) != "bbb" {
		t.Errorf("minor = %v", v)
	}
}

func TestTheFingerprintIgnoresLineEndingsAndMovesWithTheContent(t *testing.T) {
	root := t.TempDir()
	write := func(name, s string) string {
		p := filepath.Join(root, name)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write("go.mod", "module m\n")
	write("go.sum", "")
	a := write("a.go", "package a\r\n")
	one, err := fingerprint(root, []string{a})
	if err != nil {
		t.Fatal(err)
	}
	write("a.go", "package a\n")
	if two, _ := fingerprint(root, []string{a}); two != one {
		t.Error("CRLF and LF must hash the same")
	}
	write("a.go", "package b\n")
	if three, _ := fingerprint(root, []string{a}); three == one {
		t.Error("another content must hash differently")
	}
}

func TestOnlyTheModuleOwnSourcesCount(t *testing.T) {
	std := pkg{Dir: "/go/src/fmt", Standard: true, GoFiles: []string{"a.go"}}
	dep := pkg{Dir: "/mod/x", Module: &struct{ Main bool }{false}, GoFiles: []string{"a.go"}}
	own := pkg{Dir: "/repo/p", Module: &struct{ Main bool }{true}, GoFiles: []string{"a.go"}, EmbedFiles: []string{"l/en.json"}}
	if len(std.files())+len(dep.files()) != 0 || len(own.files()) != 2 {
		t.Errorf("std=%v dep=%v own=%v", std.files(), dep.files(), own.files())
	}
}
