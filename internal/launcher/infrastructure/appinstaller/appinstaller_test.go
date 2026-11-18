package appinstaller

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
)

func app(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestInstallDownloadsTheAppThenRegistersIt(t *testing.T) {
	srv := app(t, "BINARY")
	root := t.TempDir()
	var registered []string
	in := Installer{Client: srv.Client(), Root: root, Register: func(root string, l domain.Listing) error {
		registered = append(registered, l.ID)
		return nil
	}}
	l := domain.Listing{ID: "books", URL: srv.URL + "/books.app", Size: 6}

	if err := in.Install(l); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "applications", "books.app")); string(got) != "BINARY" {
		t.Errorf("content %q", got)
	}
	if len(registered) != 1 || registered[0] != "books" {
		t.Errorf("registered: %v", registered)
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "books.app.tmp")); err == nil {
		t.Error("no temporary file is left")
	}
}

func TestInstallRefusesAnIncompleteOrMissingDownloadAndLeavesNothing(t *testing.T) {
	srv := app(t, "BIN")
	root := t.TempDir()
	in := Installer{Client: srv.Client(), Root: root, Register: func(string, domain.Listing) error { return nil }}

	if err := in.Install(domain.Listing{ID: "a", URL: srv.URL + "/a.app", Size: 99}); err == nil {
		t.Error("the size does not match: the download was cut")
	}
	if err := in.Install(domain.Listing{ID: "b", URL: srv.URL + "/missing"}); err == nil {
		t.Error("a missing file")
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "a.app")); err == nil {
		t.Error("nothing is written for a refused download")
	}
}

func TestInstallReportsDiskAndRegistrationErrors(t *testing.T) {
	srv := app(t, "BIN")
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
	os.MkdirAll(filepath.Join(root, "applications", "x.app", "sub"), 0o755)
	in.Root = root
	if err := in.Install(domain.Listing{ID: "x", URL: srv.URL}); err == nil {
		t.Error("an app that cannot be replaced")
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "x.app.tmp")); err == nil {
		t.Error("the temporary file is removed on failure")
	}
}
