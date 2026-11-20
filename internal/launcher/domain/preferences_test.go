package domain

import (
	"reflect"
	"testing"
)

func TestPreferencesAreNormalizedIntoTheirBounds(t *testing.T) {
	cases := []struct {
		name string
		in   Preferences
		want Preferences
	}{
		{"zero is the default", Preferences{}, DefaultPreferences()},
		{"too small an icon", Preferences{IconSize: 10}, Preferences{Grid: DefaultGrid, IconSize: MinIconSize}},
		{"too big an icon", Preferences{IconSize: 999}, Preferences{Grid: DefaultGrid, IconSize: MaxIconSize}},
		{"the grid is clamped", Preferences{Grid: Grid{Columns: 99, Rows: -3}, IconSize: 120},
			Preferences{Grid: Grid{Columns: MaxSide, Rows: MinSide}, IconSize: 120}},
		{"a background is a file name", Preferences{Image: "  sea.png "}, Preferences{Grid: DefaultGrid, IconSize: DefaultIconSize, Image: "sea.png"}},
		{"never a path", Preferences{Image: "../secret.png"}, DefaultPreferences()},
		{"not a Windows path either", Preferences{Image: "a\\b.png"}, DefaultPreferences()},
		{"nor a parent", Preferences{Image: ".."}, DefaultPreferences()},
	}
	for _, c := range cases {
		if got := c.in.Normalized(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}
