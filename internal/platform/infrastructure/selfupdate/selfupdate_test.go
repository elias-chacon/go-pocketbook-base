package selfupdate

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go-pocketbook-base/internal/platform/domain"
)

type route map[string][]byte

func (r route) Do(req *http.Request) (*http.Response, error) {
	body, ok := r[req.URL.String()]
	if !ok {
		return &http.Response{StatusCode: 404, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, _ := zw.Create(name)
		w.Write([]byte(content))
	}
	zw.Close()
	return buf.Bytes()
}

func sum(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func fakeRelease(t *testing.T, z []byte) route {
	t.Helper()
	return route{
		"https://api.github.com/repos/o/r/releases/latest": []byte(`{"tag_name":"v1.2.3","assets":[
			{"name":"catalog.json","browser_download_url":"https://x/catalog.json"},
			{"name":"hello.zip","browser_download_url":"https://x/hello.zip"}]}`),
		"https://x/catalog.json": []byte(`{"apps":[{"id":"hello","version":"v1.2.3","asset":"hello.zip","size":` +
			strconv.Itoa(len(z)) + `,"sha256":"` + sum(z) + `","app_sha256":"` + sum([]byte("ELFbinary")) + `"},{"id":"gone","version":"1.0.0","asset":"gone.zip"}]}`),
		"https://x/hello.zip": z,
	}
}

func TestLatestFindsTheAppInTheCatalogOfTheLatestRelease(t *testing.T) {
	z := zipOf(t, map[string]string{"hello.app": "\x7fELFbinary"})
	r := fakeRelease(t, z)
	up, err := Source{Client: r, Owner: "o", Repo: "r"}.Latest("hello")
	if err != nil || up.Version != "1.2.3" || up.URL != "https://x/hello.zip" || up.SHA256 != sum(z) || up.AppSHA256 != sum([]byte("ELFbinary")) {
		t.Fatalf("update = %+v, %v", up, err)
	}
	if _, err := (Source{Client: r, Owner: "o", Repo: "r"}).Latest("nope"); err == nil {
		t.Error("an app that is not in the release has no update")
	}
	if _, err := (Source{Client: r, Owner: "o", Repo: "r"}).Latest("gone"); err == nil {
		t.Error("a catalog entry whose zip is not in the release is an error")
	}
	if _, err := (Source{Client: route{}, Owner: "o", Repo: "r"}).Latest("hello"); err == nil {
		t.Error("no release must be an error")
	}
}

func TestInstallReplacesTheFileAndLeavesNoTemporaryOne(t *testing.T) {
	z := zipOf(t, map[string]string{"hello.app": "\x7fELFnew", "app.json": "{}"})
	exe := filepath.Join(t.TempDir(), "applications", "hello.app")
	os.MkdirAll(filepath.Dir(exe), 0o755)
	os.WriteFile(exe, []byte("\x7fELFold"), 0o755)
	r := fakeRelease(t, z)
	in := Installer{Client: r, Exe: exe}
	up := domain.Update{URL: "https://x/hello.zip", Size: int64(len(z)), SHA256: sum(z)}
	if err := in.Install("hello", up); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "\x7fELFnew" {
		t.Errorf("file = %q", got)
	}
	if _, err := os.Stat(exe + ".new"); err == nil {
		t.Error("the temporary file must be gone")
	}
}

func TestInstallRefusesWhatIsCorruptIncompleteOrNotAProgramAndKeepsTheOldApp(t *testing.T) {
	good := zipOf(t, map[string]string{"hello.app": "\x7fELFnew"})
	bad := zipOf(t, map[string]string{"hello.app": "#!/bin/sh"})
	empty := zipOf(t, map[string]string{"other.app": "\x7fELF"})
	cases := map[string]struct {
		zip []byte
		up  domain.Update
	}{
		"wrong sum":     {good, domain.Update{URL: "https://x/hello.zip", SHA256: sum([]byte("other"))}},
		"wrong size":    {good, domain.Update{URL: "https://x/hello.zip", Size: 1}},
		"not elf":       {bad, domain.Update{URL: "https://x/hello.zip"}},
		"no app":        {empty, domain.Update{URL: "https://x/hello.zip"}},
		"not a zip":     {[]byte("junk"), domain.Update{URL: "https://x/hello.zip"}},
		"wrong app sum": {good, domain.Update{URL: "https://x/hello.zip", AppSHA256: sum([]byte("other"))}},
		"missing file":  {good, domain.Update{URL: "https://x/none.zip"}},
	}
	for name, c := range cases {
		exe := filepath.Join(t.TempDir(), "hello.app")
		os.WriteFile(exe, []byte("old"), 0o755)
		in := Installer{Client: route{"https://x/hello.zip": c.zip}, Exe: exe}
		if err := in.Install("hello", c.up); err == nil {
			t.Errorf("%s: must fail", name)
		}
		if got, _ := os.ReadFile(exe); string(got) != "old" {
			t.Errorf("%s: the old app must stay, got %q", name, got)
		}
	}
}
