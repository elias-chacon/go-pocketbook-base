package appinstaller

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
)

const manifestOf = `{"id":"%s","title":{"pt-BR":"T"},"description":{"pt-BR":"D"}}`

func manifest(id string) string { return strings.Replace(manifestOf, "%s", id, 1) }

// makeZip builds a zip with the given files (name -> content).
func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(content))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func serve(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func listing(srv *httptest.Server, id string, zipData []byte) domain.Listing {
	sum := sha256.Sum256(zipData)
	return domain.Listing{ID: id, URL: srv.URL + "/" + id + ".zip", Size: int64(len(zipData)), SHA256: hex.EncodeToString(sum[:])}
}

func TestInstallDownloadsUnzipsAndRegistersWithTheManifestOfTheZip(t *testing.T) {
	zipData := makeZip(t, map[string]string{"books.app": "BINARY", "app.json": manifest("books"), "../evil.txt": "ignored"})
	srv := serve(t, zipData)
	root := t.TempDir()
	var registered domain.Listing
	in := Installer{Client: srv.Client(), Root: root, Register: func(root string, l domain.Listing) error {
		registered = l
		return nil
	}}

	if err := in.Install(listing(srv, "books", zipData)); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "applications", "books.app")); string(got) != "BINARY" {
		t.Errorf("content %q", got)
	}
	if registered.ID != "books" || string(registered.Manifest) != manifest("books") {
		t.Errorf("registered with the manifest of the zip: %+v", registered)
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "books.app.tmp")); err == nil {
		t.Error("no temporary file is left")
	}
	if _, err := os.Stat(filepath.Join(root, "evil.txt")); err == nil {
		t.Error("nothing is written from the zip's own paths")
	}
}

func TestInstallRefusesWhatDoesNotMatchTheCatalog(t *testing.T) {
	good := makeZip(t, map[string]string{"a.app": "BIN", "app.json": manifest("a")})
	srv := serve(t, good)
	root := t.TempDir()
	in := Installer{Client: srv.Client(), Root: root, Register: func(string, domain.Listing) error { return nil }}

	l := listing(srv, "a", good)
	l.Size++
	if err := in.Install(l); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Errorf("a cut download: %v", err)
	}
	l = listing(srv, "a", good)
	l.SHA256 = strings.Repeat("0", 64)
	if err := in.Install(l); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Errorf("a corrupted download: %v", err)
	}
	if err := in.Install(domain.Listing{ID: "a", URL: srv.URL + "/missing"}); err == nil {
		t.Error("a missing file")
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "a.app")); err == nil {
		t.Error("nothing is written for a refused download")
	}
}

func TestInstallRefusesBadZips(t *testing.T) {
	cases := map[string][]byte{
		"not a zip":            []byte("plain text"),
		"no app":               makeZip(t, map[string]string{"app.json": manifest("x")}),
		"no manifest":          makeZip(t, map[string]string{"x.app": "B"}),
		"a manifest that lies": makeZip(t, map[string]string{"x.app": "B", "app.json": manifest("other")}),
		"a broken manifest":    makeZip(t, map[string]string{"x.app": "B", "app.json": "{"}),
	}
	for name, data := range cases {
		srv := serve(t, data)
		in := Installer{Client: srv.Client(), Root: t.TempDir(), Register: func(string, domain.Listing) error { return nil }}
		if err := in.Install(domain.Listing{ID: "x", URL: srv.URL}); err == nil {
			t.Errorf("%s must be refused", name)
		}
	}
}

func TestInstallReportsDiskAndRegistrationErrors(t *testing.T) {
	data := makeZip(t, map[string]string{"a.app": "BIN", "app.json": manifest("a")})
	srv := serve(t, data)
	boom := errors.New("no icon")
	in := Installer{Client: srv.Client(), Root: t.TempDir(), Register: func(string, domain.Listing) error { return boom }}
	if err := in.Install(domain.Listing{ID: "a", URL: srv.URL}); err != boom {
		t.Errorf("registration error: %v", err)
	}

	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	in.Root = file
	if err := in.Install(domain.Listing{ID: "a", URL: srv.URL}); err == nil {
		t.Error("a storage that cannot be written")
	}

	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "applications", "a.app", "sub"), 0o755)
	in.Root = root
	if err := in.Install(domain.Listing{ID: "a", URL: srv.URL}); err == nil {
		t.Error("an app that cannot be replaced")
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "a.app.tmp")); err == nil {
		t.Error("the temporary file is removed on failure")
	}
}
