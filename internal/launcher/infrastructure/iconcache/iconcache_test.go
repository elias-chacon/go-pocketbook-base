package iconcache

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
)

func pngBytes(t *testing.T, gray uint8) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = gray
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func server(t *testing.T, body []byte, hits *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if r.URL.Path == "/missing.png" {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTheIconIsDownloadedOnceAndThenComesFromTheCache(t *testing.T) {
	var hits int32
	srv := server(t, pngBytes(t, 40), &hits)
	c := Cache{Client: srv.Client(), Dir: filepath.Join(t.TempDir(), "cache", "icons")}
	l := domain.Listing{ID: "hello", Version: "1.0.0", IconURL: srv.URL + "/hello.png"}

	img, err := c.Image(l)
	if err != nil || img.Bounds().Dx() != 8 {
		t.Fatalf("%v %v", img, err)
	}
	if _, err := c.Image(l); err != nil || hits != 1 {
		t.Errorf("the second one is the cache's: hits=%d err=%v", hits, err)
	}
	if _, err := os.Stat(filepath.Join(c.Dir, "hello.png")); err != nil {
		t.Errorf("kept on disk: %v", err)
	}

	l.IconURL = srv.URL + "/v2/hello.png" // the next release publishes it under another address
	c.Image(l)
	if hits != 2 {
		t.Errorf("a new address is a new icon: %d", hits)
	}
	files, _ := os.ReadDir(c.Dir)
	if len(files) != 2 { // hello.png and hello.src: the new picture took the place of the old
		t.Errorf("one picture per app, got %d files", len(files))
	}
}

func TestPruneDropsWhatTheCatalogNoLongerListsAndTheOldNames(t *testing.T) {
	c := Cache{Dir: t.TempDir()}
	for _, n := range []string{"hello.png", "hello.src", "gone.png", "gone.src", "hello.url", "hello-1.0.0.png", "notes.txt"} {
		os.WriteFile(filepath.Join(c.Dir, n), []byte("x"), 0o644)
	}
	c.Prune(nil)
	if files, _ := os.ReadDir(c.Dir); len(files) != 7 {
		t.Errorf("an empty catalog must not empty the cache: %d", len(files))
	}
	c.Prune([]domain.Listing{{ID: "hello"}})
	var left []string
	files, _ := os.ReadDir(c.Dir)
	for _, f := range files {
		left = append(left, f.Name())
	}
	if len(left) != 3 || left[0] != "hello.png" || left[1] != "hello.src" || left[2] != "notes.txt" {
		t.Errorf("left = %v", left)
	}
}

func TestADamagedCacheFileIsFetchedAgain(t *testing.T) {
	var hits int32
	srv := server(t, pngBytes(t, 40), &hits)
	c := Cache{Client: srv.Client(), Dir: t.TempDir()}
	os.WriteFile(filepath.Join(c.Dir, "hello.png"), []byte("garbage"), 0o644)
	os.WriteFile(filepath.Join(c.Dir, "hello.src"), []byte(srv.URL+"/x.png"), 0o644)

	if _, err := c.Image(domain.Listing{ID: "hello", Version: "1", IconURL: srv.URL + "/x.png"}); err != nil || hits != 1 {
		t.Errorf("hits=%d err=%v", hits, err)
	}
}

func TestMissingBrokenOrUnreachableIconsAreErrors(t *testing.T) {
	var hits int32
	srv := server(t, []byte("not a png"), &hits)
	c := Cache{Client: srv.Client(), Dir: t.TempDir()}

	if _, err := c.Image(domain.Listing{ID: "a"}); err != ErrNoIcon {
		t.Errorf("an app without an icon: %v", err)
	}
	if _, err := c.Image(domain.Listing{ID: "a", Version: "1", IconURL: srv.URL + "/missing.png"}); err == nil {
		t.Error("a 404")
	}
	if _, err := c.Image(domain.Listing{ID: "a", Version: "1", IconURL: srv.URL + "/a.png"}); err == nil {
		t.Error("something that is not a PNG")
	}
	if _, err := os.Stat(filepath.Join(c.Dir, "a.png")); err == nil {
		t.Error("a broken download is not kept")
	}
}

func TestAnUnwritableCacheStillAnswers(t *testing.T) {
	var hits int32
	srv := server(t, pngBytes(t, 40), &hits)
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	c := Cache{Client: srv.Client(), Dir: filepath.Join(file, "sub")} // a folder under a file cannot be made

	if img, err := c.Image(domain.Listing{ID: "a", Version: "1", IconURL: srv.URL + "/a.png"}); err != nil || img == nil {
		t.Errorf("%v %v", img, err)
	}
}
