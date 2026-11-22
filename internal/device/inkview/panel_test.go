package inkview

import "testing"

func TestARectangleOutsideItsPictureIsNotDrawn(t *testing.T) {
	const stride, rows = 40, 10 // 10 pixels of 4 bytes per row, 10 rows
	n := stride * rows
	for _, tc := range []struct {
		x, y, w, h int
		want       bool
	}{
		{0, 0, 10, 10, true}, {2, 3, 5, 7, true}, {0, 5, 10, 5, true},
		{0, 5, 10, 6, false}, {-1, 0, 5, 5, false}, {0, 0, 11, 5, false}, {0, 0, 0, 5, false}, {5, 12, 2, 2, false},
	} {
		if got := inside(n, stride, 4, tc.x, tc.y, tc.w, tc.h); got != tc.want {
			t.Errorf("%+v: %v, want %v", tc, got, tc.want)
		}
	}
	if UpdateA2.String() != "a2" || UpdateMode(42).String() != "unknown" || UpdateModes != 7 {
		t.Error("the update modes have names")
	}
}
