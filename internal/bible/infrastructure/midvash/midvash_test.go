package midvash

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-pocketbook-base/internal/bible/domain"
)

const booksJSON = `{"data":[
 {"id":1,"name":{"en":"Genesis","pt-br":"Gênesis"},"slug":{"en":"genesis","pt-br":"genesis"},"chapters":50,"testament":"old"},
 {"id":43,"name":{"en":"John","pt-br":"João"},"slug":{"en":"john","pt-br":"joao"},"chapters":21,"testament":"new"}]}`

const versionsJSON = `{"data":[
 {"slug":"kjv","name":"King James Version","shortName":"KJV","language":"en","totalBooks":66,"totalChapters":1189},
 {"slug":"almeida-livre","name":"Bíblia Livre","shortName":"BL","language":"pt-br","totalBooks":66,"totalChapters":1189}]}`

func server(t *testing.T, requests *[]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.URL.RequestURI())
		if r.Header.Get("Authorization") != "" {
			t.Error("a public API gets no Authorization header")
		}
		switch r.URL.Path {
		case "/v1/books":
			fmt.Fprint(w, booksJSON)
		case "/v1/versions":
			fmt.Fprint(w, versionsJSON)
		case "/v1/passages":
			var items []string
			for _, ref := range strings.Split(r.URL.Query().Get("refs"), ",") {
				switch ref {
				case "genesis 1":
					items = append(items, `{"book":"genesis","chapter":1,"verse":1,"verses":["No princípio","Era sem forma "]}`)
				case "john 3":
					items = append(items, `{"book":"john","chapter":3,"verse":1,"verses":["a","b","c"]}`)
				case "john 99":
					items = append(items, `{"book":"john","chapter":99,"verse":1,"verses":[]}`)
				}
			}
			fmt.Fprintf(w, `{"data":[%s]}`, strings.Join(items, ","))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTheCatalogReadsBooksAndVersions(t *testing.T) {
	var reqs []string
	srv := server(t, &reqs)
	c := Client{HTTP: srv.Client(), Base: srv.URL + "/v1"}

	books, err := c.Books()
	if err != nil || len(books) != 2 {
		t.Fatalf("books: %v %v", books, err)
	}
	if books[1].ID != 43 || books[1].Slug != "john" || books[1].Name("pt-BR") != "João" || !books[1].New || books[0].New || books[0].Chapters != 50 {
		t.Errorf("john: %+v", books[1])
	}
	versions, err := c.Versions()
	if err != nil || len(versions) != 2 || versions[1].Language != "pt-br" || versions[0].Chapters != 1189 || versions[0].ShortName != "KJV" {
		t.Fatalf("versions: %+v %v", versions, err)
	}
}

func TestChaptersComeInBatchesAndSkipWhatTheVersionLacks(t *testing.T) {
	var reqs []string
	srv := server(t, &reqs)
	c := Client{HTTP: srv.Client(), Base: srv.URL + "/v1"}
	books := []domain.Book{{ID: 1, Slug: "genesis"}, {ID: 43, Slug: "john"}}

	got, err := c.Chapters("kjv", books, []domain.ChapterRef{domain.Chapter(1, 1), domain.Chapter(43, 3), domain.Chapter(43, 99)})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Book != 1 || got[0].Version != "kjv" || len(got[0].Verses) != 2 || got[0].Verses[1] != (domain.Verse{Number: 2, Text: "Era sem forma"}) {
		t.Fatalf("chapters: %+v", got)
	}
	if len(reqs) != 1 || !strings.Contains(reqs[0], "version=kjv") || !strings.Contains(reqs[0], "genesis+1%2Cjohn+3%2Cjohn+99") {
		t.Errorf("one request with the three chapters: %v", reqs)
	}

	reqs = nil
	var many []domain.ChapterRef
	for i := 0; i < BatchSize*2+3; i++ {
		many = append(many, domain.ChapterRef{Book: 43, Chapter: 3})
	}
	if _, err := c.Chapters("kjv", books, many); err != nil || len(reqs) != 3 {
		t.Errorf("%d chapters need 3 requests: %v %v", len(many), len(reqs), err)
	}
	if _, err := c.Chapters("kjv", books, []domain.ChapterRef{domain.Chapter(7, 1)}); err == nil {
		t.Error("an unknown book is an error")
	}
}

func TestAServerErrorIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":"NOT_FOUND","message":"nope"}}`, 404)
	}))
	defer srv.Close()
	c := Client{HTTP: srv.Client(), Base: srv.URL}
	if _, err := c.Books(); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("err = %v", err)
	}
	if _, err := c.Versions(); err == nil {
		t.Error("versions must fail too")
	}
	if _, err := c.Chapters("kjv", []domain.Book{{ID: 1, Slug: "genesis"}}, []domain.ChapterRef{domain.Chapter(1, 1)}); err == nil {
		t.Error("chapters must fail too")
	}
}
