package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go-pocketbook-base/internal/launcher/domain"
)

const helloManifest = `{"id":"hello","title":{"pt-BR":"Olá","en":"Hello"},"description":{"pt-BR":"Vitrine","en":"Showcase"}}`

// server serves a release whose catalog.json lists hello (valid), books (a
// manifest of another id), notes (its zip is not in the release) and a stray file.
func server(t *testing.T, hits *int32, withCatalog bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if r.Header.Get("User-Agent") == "" {
			t.Error("GitHub requires a User-Agent")
		}
		catalog := ""
		if withCatalog {
			catalog = fmt.Sprintf(`{"name":"catalog.json","browser_download_url":"%s/dl/catalog.json"},`, srv.URL)
		}
		fmt.Fprintf(w, `{"tag_name":"v1.2.3","assets":[%[2]s
			{"name":"hello.zip","browser_download_url":"%[1]s/dl/hello.zip"},
			{"name":"hello.png","browser_download_url":"%[1]s/dl/hello.png"},
			{"name":"books.zip","browser_download_url":"%[1]s/dl/books.zip"},
			{"name":"stray.app","browser_download_url":"%[1]s/dl/stray"},
			{"name":"SHA256SUMS","browser_download_url":"%[1]s/dl/sums"}]}`, srv.URL, catalog)
	})
	mux.HandleFunc("/dl/catalog.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"apps":[
			{"id":"hello","version":"v1.2.3","asset":"hello.zip","icon":"hello.png","size":4096,"sha256":"abc","manifest":%[1]s},
			{"id":"books","version":"1.0.0","asset":"books.zip","size":1,"sha256":"d","manifest":%[1]s},
			{"id":"notes","version":"1.0.0","asset":"notes.zip","size":1,"sha256":"e","manifest":%[1]s},
			{"id":"broken","version":"1.0.0","asset":"hello.zip","size":1,"sha256":"f","manifest":{"id":"broken"}}]}`, helloManifest)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestTheCatalogListsTheAppsOfTheLatestReleaseInTheLanguage(t *testing.T) {
	var hits int32
	srv := server(t, &hits, true)
	c := &Catalog{Client: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}

	list, err := c.List("en")
	if err != nil || len(list) != 1 {
		t.Fatalf("only hello is valid (books has another id, notes is not in the release, broken has no title): %+v %v", list, err)
	}
	l := list[0]
	if l.ID != "hello" || l.Title != "Hello" || l.Description != "Showcase" || l.Version != "1.2.3" {
		t.Errorf("%+v", l)
	}
	if !strings.HasSuffix(l.IconURL, "/dl/hello.png") {
		t.Errorf("the icon of the release: %q", l.IconURL)
	}
	if !strings.HasSuffix(l.URL, "/dl/hello.zip") || l.Size != 4096 || l.SHA256 != "abc" || string(l.Manifest) != helloManifest {
		t.Errorf("download data: %+v", l)
	}

	pt, _ := c.List("pt-BR")
	if len(pt) != 1 || pt[0].Title != "Olá" {
		t.Errorf("another language is another answer: %+v", pt)
	}
}

func TestAReleaseWithoutACatalogIsAnError(t *testing.T) {
	var hits int32
	srv := server(t, &hits, false)
	c := &Catalog{Client: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	if _, err := c.List("en"); err == nil || !strings.Contains(err.Error(), "catalog.json") {
		t.Errorf("%v", err)
	}
}

func TestTheAnswerIsKeptForAWhile(t *testing.T) {
	var hits int32
	srv := server(t, &hits, true)
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

	badIndex := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "latest") {
			fmt.Fprintf(w, `{"tag_name":"v1","assets":[{"name":"catalog.json","browser_download_url":"http://%s/idx"}]}`, r.Host)
			return
		}
		fmt.Fprint(w, "not json")
	}))
	defer badIndex.Close()
	if _, err := (&Catalog{Client: badIndex.Client(), Owner: "o", Repo: "r", BaseURL: badIndex.URL}).List("en"); err == nil ||
		!strings.Contains(err.Error(), "catalog.json") {
		t.Errorf("a catalog.json that is not JSON: %v", err)
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

func TestAppAssetsAreZipsAndIconsArePNGs(t *testing.T) {
	if AppAsset("hello") != "hello.zip" || IconAsset("hello") != "hello.png" {
		t.Error(AppAsset("hello"), IconAsset("hello"))
	}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestTheCatalogReportsOneStepForTheReleaseAndOneForTheCatalog(t *testing.T) {
	var hits int32
	srv := server(t, &hits, true)
	var p domain.Progress
	p.Begin(2)
	c := &Catalog{Client: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL, Progress: &p}
	if _, err := c.List("en"); err != nil {
		t.Fatal(err)
	}
	if f, _ := p.State(); f != 1 {
		t.Errorf("after the release and the catalog.json the 2 steps are done, got %v", f)
	}
}
