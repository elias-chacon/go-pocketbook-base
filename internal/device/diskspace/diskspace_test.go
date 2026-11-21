package diskspace

import (
	"runtime"
	"testing"
)

func TestOfTellsTheSpaceOfAVolumeWhereItCanBeKnown(t *testing.T) {
	s, ok := Of(t.TempDir())
	if runtime.GOOS != "linux" {
		if ok {
			t.Error("only Linux can tell")
		}
		return
	}
	if !ok || s.Total == 0 || s.Free > s.Total {
		t.Errorf("%+v %v", s, ok)
	}
	if _, ok := Of("/no/such/folder/at/all"); ok {
		t.Error("a path that does not exist has no space")
	}
}
