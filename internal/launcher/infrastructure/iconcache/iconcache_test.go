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
	if _, err := os.Stat(filepath.Join(c.Dir, "hello-1.0.0.png")); err != nil {
		t.Errorf("kept on disk: %v", err)
	}

	l.Version = "1.1.0"
	c.Image(l)
	if hits != 2 {
		t.Errorf("a new version is a new icon: %d", hits)
	}
}

func TestADamagedCacheFileIsFetchedAgain(t *testing.T) {
	var hits int32
	srv := server(t, pngBytes(t, 40), &hits)
	c := Cache{Client: srv.Client(), Dir: t.TempDir()}
	os.WriteFile(filepath.Join(c.Dir, "hello-1.png"), []byte("garbage"), 0o644)

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
	if _, err := os.Stat(filepath.Join(c.Dir, "a-1.png")); err == nil {
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
