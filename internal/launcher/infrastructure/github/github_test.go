package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const helloManifest = `{"id":"hello","title":{"pt-BR":"Olá","en":"Hello"},"description":{"pt-BR":"Vitrine","en":"Showcase"}}`

// server serves a release with hello (valid), books (manifest of another id),
// notes.app without a manifest, and a stray file.
func server(t *testing.T, hits *int32) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if r.Header.Get("User-Agent") == "" {
			t.Error("GitHub requires a User-Agent")
		}
		fmt.Fprintf(w, `{"tag_name":"v1.2.3","assets":[
			{"name":"hello.app.bin","browser_download_url":"%[1]s/dl/hello.app","size":4096},
			{"name":"hello.json","browser_download_url":"%[1]s/dl/hello.json","size":10},
			{"name":"books.app.bin","browser_download_url":"%[1]s/dl/books.app","size":1},
			{"name":"books.json","browser_download_url":"%[1]s/dl/books-wrong.json","size":10},
			{"name":"notes.app.bin","browser_download_url":"%[1]s/dl/notes.app","size":1},
			{"name":"stray.app","browser_download_url":"%[1]s/dl/stray","size":1},
			{"name":"SHA256SUMS","browser_download_url":"%[1]s/dl/sums","size":1}]}`, srv.URL)
	})
	mux.HandleFunc("/dl/hello.json", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, helloManifest) })
	mux.HandleFunc("/dl/books-wrong.json", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, helloManifest) })
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestTheCatalogListsTheAppsOfTheLatestReleaseInTheLanguage(t *testing.T) {
	var hits int32
	srv := server(t, &hits)
	c := &Catalog{Client: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}

	list, err := c.List("en")
	if err != nil || len(list) != 1 {
		t.Fatalf("only hello is a valid app: %+v %v", list, err)
	}
	l := list[0]
	if l.ID != "hello" || l.Title != "Hello" || l.Description != "Showcase" || l.Version != "1.2.3" {
		t.Errorf("%+v", l)
	}
	if !strings.HasSuffix(l.URL, "/dl/hello.app") || l.Size != 4096 || string(l.Manifest) != helloManifest {
		t.Errorf("download data: %+v", l)
	}

	pt, _ := c.List("pt-BR")
	if len(pt) != 1 || pt[0].Title != "Olá" {
		t.Errorf("another language is another answer: %+v", pt)
	}
}

func TestTheAnswerIsKeptForAWhile(t *testing.T) {
	var hits int32
	srv := server(t, &hits)
	now := time.Unix(1000, 0)
	c := &Catalog{Client: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL, Now: func() time.Time { return now }}

	c.List("en")
	c.List("en")
	if hits != 1 {
		t.Errorf("the second search does not ask GitHub again: %d", hits)
	}
	now = now.Add(cacheFor + time.Second)
	c.List("en")
	if hits != 2 {
		t.Errorf("after a while it does: %d", hits)
	}
}

func TestFailuresAreReported(t *testing.T) {
	notFound := httptest.NewServer(http.NotFoundHandler())
	defer notFound.Close()
	if _, err := (&Catalog{Client: notFound.Client(), Owner: "o", Repo: "r", BaseURL: notFound.URL}).List("en"); err == nil ||
		!strings.Contains(err.Error(), "404") {
		t.Errorf("a missing release: %v", err)
	}

	garbage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>") }))
	defer garbage.Close()
	if _, err := (&Catalog{Client: garbage.Client(), Owner: "o", Repo: "r", BaseURL: garbage.URL}).List("en"); err == nil {
		t.Error("an answer that is not JSON")
	}

	if _, err := (&Catalog{Client: http.DefaultClient, Owner: "o", Repo: "r", BaseURL: "http://127.0.0.1:1"}).List("en"); err == nil {
		t.Error("no network")
	}
	if _, err := (&Catalog{Client: http.DefaultClient, Owner: "o", Repo: "r", BaseURL: "://bad"}).List("en"); err == nil {
		t.Error("a bad URL")
	}
}

func TestGetRefusesWhatIsTooBigOrNotOK(t *testing.T) {
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("x", 100))
	}))
	defer big.Close()
	if _, err := Get(big.Client(), big.URL, 50); err == nil || !strings.Contains(err.Error(), "larger") {
		t.Errorf("limit: %v", err)
	}
	if data, err := Get(big.Client(), big.URL, 100); err != nil || len(data) != 100 {
		t.Errorf("exactly the limit is fine: %d %v", len(data), err)
	}
}

func TestTheDefaultBaseURLIsGitHub(t *testing.T) {
	var got string
	c := &Catalog{Client: doerFunc(func(r *http.Request) (*http.Response, error) {
		got = r.URL.String()
		return nil, fmt.Errorf("stop")
	}), Owner: "o", Repo: "r"}
	c.List("en")
	if got != "https://api.github.com/repos/o/r/releases/latest" {
		t.Errorf("%s", got)
	}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }
