package catalogcache

import (
	"errors"
	"path/filepath"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
)

type inner struct {
	list []domain.Listing
	err  error
}

func (i *inner) List(string) ([]domain.Listing, error) { return i.list, i.err }

const manifest = `{"id":"bible","title":{"pt-BR":"Bíblia","en":"Bible"},"description":{"pt-BR":"Leia","en":"Read"}}`

func listing() domain.Listing {
	return domain.Listing{ID: "bible", Title: "Bíblia", Version: "1.2.3", URL: "u", AppSHA256: "abc", Manifest: []byte(manifest)}
}

func TestTheAnswerIsKeptAndGivenWhenTheNetworkFails(t *testing.T) {
	src := &inner{list: []domain.Listing{listing()}}
	c := &Catalog{Inner: src, Path: filepath.Join(t.TempDir(), "cache", "catalog.json")}
	if _, err := c.Cached("pt-BR"); err != ErrNothingKept {
		t.Fatalf("nothing kept yet: %v", err)
	}
	if got, err := c.List("pt-BR"); err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	src.list, src.err = nil, errors.New("offline")
	got, err := c.List("en")
	if err != nil || len(got) != 1 || got[0].Title != "Bible" || got[0].AppSHA256 != "abc" || got[0].Version != "1.2.3" {
		t.Errorf("offline gives the kept catalog, in the language asked: %+v %v", got, err)
	}
}

func TestWithNothingKeptTheNetworkErrorReaches(t *testing.T) {
	c := &Catalog{Inner: &inner{err: errors.New("offline")}, Path: filepath.Join(t.TempDir(), "c.json")}
	if _, err := c.List("pt-BR"); err == nil || err.Error() != "offline" {
		t.Errorf("err = %v", err)
	}
}
