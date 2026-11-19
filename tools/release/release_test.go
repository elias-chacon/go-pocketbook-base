package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go-pocketbook-base/internal/launcher/infrastructure/github"
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
	zipPath := filepath.Join(dir, win.archive())
	if filepath.Base(zipPath) != "pbgo-installer_windows_amd64.zip" {
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
	tgz := filepath.Join(dir, lin.archive())
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

func TestTheNotesGiveTheInstallCommandOfEachSystem(t *testing.T) {
	text := notes("https://github.com/o/r/releases/download/v1.2.3")
	for _, want := range []string{
		"Invoke-WebRequest https://github.com/o/r/releases/download/v1.2.3/pbgo-installer_windows_amd64.zip",
		"Expand-Archive", `.\pbgo-installer\pbgo-installer.exe`,
		"curl -fsSL https://github.com/o/r/releases/download/v1.2.3/pbgo-installer_linux_amd64.tar.gz | tar xz && ./pbgo-installer",
		"curl -fsSL https://github.com/o/r/releases/download/v1.2.3/pbgo-installer_darwin_${arch}.tar.gz | tar xz",
		"SHA256SUMS", "Executar assim mesmo",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the notes lack %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "%!") {
		t.Errorf("a formatting mistake:\n%s", text)
	}
}

func TestPackageAppZipsTheBinaryWithItsManifestAndDescribesIt(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "dist", "hello.app"), "BINARY")
	manifest := `{"id":"hello","title":{"pt-BR":"Olá"},"description":{"pt-BR":"D"}}`
	write(t, filepath.Join(root, "apps", "hello", "app.json"), manifest)
	out := filepath.Join(root, "out")
	os.MkdirAll(out, 0o755)

	entry, err := packageApp(filepath.Join(root, "dist"), filepath.Join(root, "apps"), out, "hello", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != "hello" || entry.Asset != "hello.zip" || entry.Version != "1.2.3" || string(entry.Manifest) != manifest {
		t.Errorf("%+v", entry)
	}

	data, err := os.ReadFile(filepath.Join(out, "hello.zip"))
	if err != nil || int64(len(data)) != entry.Size {
		t.Fatalf("size %d vs %d: %v", len(data), entry.Size, err)
	}
	if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != entry.SHA256 {
		t.Error("the catalog's SHA-256 is the zip's")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) != 2 {
		t.Fatalf("%v %v", err, zr)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		body, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(body)
		if f.Name == "hello.app" && f.Mode()&0o111 == 0 {
			t.Error("the app stays executable")
		}
	}
	if got["hello.app"] != "BINARY" || got["app.json"] != manifest {
		t.Errorf("%v", got)
	}

	again, _ := packageApp(filepath.Join(root, "dist"), filepath.Join(root, "apps"), out, "hello", "1.2.3")
	if again.SHA256 != entry.SHA256 {
		t.Error("the same build gives the same zip")
	}
}

func TestPackageAppReportsWhatIsMissing(t *testing.T) {
	root := t.TempDir()
	if _, err := packageApp(root, root, root, "x", "1"); err == nil {
		t.Error("no built app")
	}
	write(t, filepath.Join(root, "x.app"), "B")
	if _, err := packageApp(root, filepath.Join(root, "apps"), root, "x", "1"); err == nil || !strings.Contains(err.Error(), "manifest") {
		t.Errorf("no manifest: %v", err)
	}
	write(t, filepath.Join(root, "apps", "x", "app.json"), "{}")
	if _, err := packageApp(root, filepath.Join(root, "apps"), filepath.Join(root, "no-such-dir"), "x", "1"); err == nil {
		t.Error("an output folder that does not exist")
	}
}

func TestTheCatalogIsWrittenInTheFormatTheLauncherReads(t *testing.T) {
	dir := t.TempDir()
	entries := []github.IndexEntry{{ID: "a", Version: "1", Asset: "a.zip", Size: 3, SHA256: "ff", Manifest: json.RawMessage(`{"id":"a"}`)}}
	if err := writeCatalog(dir, entries); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "catalog.json"))
	var index github.Index
	if err := json.Unmarshal(raw, &index); err != nil || len(index.Apps) != 1 || index.Apps[0].Asset != "a.zip" {
		t.Errorf("%v %+v", err, index)
	}
	if err := writeCatalog(filepath.Join(dir, "none"), entries); err == nil {
		t.Error("an output folder that does not exist")
	}
}
