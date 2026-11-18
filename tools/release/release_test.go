package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
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

func TestStageCopiesEachBuiltAppWithItsManifestAndTheVersion(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "dist", "hello.app"), "H")
	write(t, filepath.Join(root, "dist", "launcher.app"), "L")
	write(t, filepath.Join(root, "apps", "hello", "app.json"), `{"id":"hello"}`)
	write(t, filepath.Join(root, "apps", "launcher", "app.json"), `{"id":"launcher"}`)
	payload := filepath.Join(root, "payload")
	os.MkdirAll(payload, 0o755)

	ids, err := stage(filepath.Join(root, "dist"), filepath.Join(root, "apps"), payload, "1.2.3")
	if err != nil || len(ids) != 2 || ids[0] != "hello" {
		t.Fatalf("%v %v", ids, err)
	}
	for file, want := range map[string]string{"hello.app": "H", "launcher.json": `{"id":"launcher"}`, "VERSION": "1.2.3\n"} {
		if got, _ := os.ReadFile(filepath.Join(payload, file)); string(got) != want {
			t.Errorf("%s = %q", file, got)
		}
	}
}

func TestStageComplainsAboutMissingBuildsAndManifests(t *testing.T) {
	root := t.TempDir()
	if _, err := stage(filepath.Join(root, "none"), root, root, "1"); err == nil || !strings.Contains(err.Error(), "build.ps1") {
		t.Errorf("no builds: %v", err)
	}
	write(t, filepath.Join(root, "dist", "x.app"), "x")
	if _, err := stage(filepath.Join(root, "dist"), filepath.Join(root, "apps"), root, "1"); err == nil {
		t.Error("an app without a manifest cannot be installed")
	}
}

func TestArchivesKeepTheExecutableAndNameIt(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "bin")
	write(t, exe, "BINARY")

	win := target{"windows", "amd64"}
	zipPath := filepath.Join(dir, win.archive("1.0.0"))
	if filepath.Base(zipPath) != "pbgo-installer_1.0.0_windows_amd64.zip" {
		t.Errorf("name: %s", zipPath)
	}
	if err := writeArchive(zipPath, win, win.exe(), exe); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil || len(zr.File) != 1 || zr.File[0].Name != "pbgo-installer.exe" {
		t.Fatalf("%v %v", err, zr)
	}
	zr.Close()

	lin := target{"linux", "amd64"}
	tgz := filepath.Join(dir, lin.archive("1.0.0"))
	if err := writeArchive(tgz, lin, lin.exe(), exe); err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(tgz)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	h, err := tr.Next()
	if err != nil || h.Name != "pbgo-installer" || h.Mode != 0o755 {
		t.Fatalf("%+v %v", h, err)
	}
	if body, _ := io.ReadAll(tr); string(body) != "BINARY" {
		t.Errorf("content %q", body)
	}
	if err := writeArchive(filepath.Join(dir, "x.zip"), win, "x", filepath.Join(dir, "missing")); err == nil {
		t.Error("a missing executable is an error")
	}
}

func TestChecksumsListEveryFileSorted(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "b.txt"), "b")
	write(t, filepath.Join(dir, "a.txt"), "a")
	if err := checksums(dir, []string{"b.txt", "a.txt"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "  a.txt") ||
		!strings.HasPrefix(lines[0], "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb") {
		t.Errorf("%v", lines)
	}
	if err := checksums(dir, []string{"nope"}); err == nil {
		t.Error("a missing file is an error")
	}
}

func TestTargetsCoverTheThreeSystems(t *testing.T) {
	seen := map[string]bool{}
	for _, tg := range targets {
		seen[tg.OS] = true
	}
	if !seen["windows"] || !seen["linux"] || !seen["darwin"] {
		t.Errorf("%v", targets)
	}
	if (target{"linux", "amd64"}).exe() != "pbgo-installer" {
		t.Error("no extension outside Windows")
	}
}
