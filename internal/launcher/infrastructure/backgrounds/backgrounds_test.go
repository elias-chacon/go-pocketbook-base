package backgrounds

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNamesListsOnlyPicturesSorted(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.PNG", "a.jpg", "c.bmp", "d.gif", "e.jpeg", "notes.txt", ".hidden.png"} {
		os.WriteFile(filepath.Join(dir, name), nil, 0o644)
	}
	os.MkdirAll(filepath.Join(dir, "folder.png"), 0o755)

	want := []string{"a.jpg", "b.PNG", "c.bmp", "d.gif", "e.jpeg"}
	if got := Names(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("%v, want %v", got, want)
	}
	if got := Names(filepath.Join(dir, "missing")); got != nil {
		t.Errorf("a missing folder: %v", got)
	}
	if Dir("/mnt/ext1") != filepath.Join("/mnt/ext1", "applications", "backgrounds") {
		t.Error(Dir("/mnt/ext1"))
	}
}
