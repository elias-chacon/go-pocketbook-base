package geom

import "testing"

func TestRectBasics(t *testing.T) {
	r := R(10, 20, 100, 50)
	if r.Right() != 110 || r.Bottom() != 70 {
		t.Errorf("edges: %v %v", r.Right(), r.Bottom())
	}
	if c := r.Center(); c.X != 60 || c.Y != 45 {
		t.Errorf("center = %+v", c)
	}
	if s := r.Size(); s.W != 100 || s.H != 50 {
		t.Errorf("size = %+v", s)
	}
}

func TestContainsExcludesRightAndBottomEdges(t *testing.T) {
	r := R(0, 0, 10, 10)
	for _, c := range []struct {
		p    Point
		want bool
	}{
		{Point{0, 0}, true}, {Point{9.99, 9.99}, true},
		{Point{10, 5}, false}, {Point{5, 10}, false}, {Point{-0.1, 5}, false},
	} {
		if got := r.Contains(c.p); got != c.want {
			t.Errorf("Contains(%+v) = %v", c.p, got)
		}
	}
}

func TestInset(t *testing.T) {
	if got := R(0, 0, 100, 60).Inset(XY(10, 20)); got != R(20, 10, 60, 40) {
		t.Errorf("Inset = %+v", got)
	}
	if got := R(0, 0, 10, 10).Inset(All(20)); got.W != 0 || got.H != 0 {
		t.Errorf("over-inset must not go negative: %+v", got)
	}
	i := Insets{1, 2, 3, 4}
	if i.Horizontal() != 6 || i.Vertical() != 4 {
		t.Errorf("sums = %v %v", i.Horizontal(), i.Vertical())
	}
}

func TestMinMaxClamp(t *testing.T) {
	if Min(1, 2) != 1 || Max(1, 2) != 2 || Min(3, 3) != 3 {
		t.Error("Min/Max")
	}
	for _, c := range [][4]float64{{5, 0, 10, 5}, {-1, 0, 10, 0}, {11, 0, 10, 10}} {
		if got := Clamp(c[0], c[1], c[2]); got != c[3] {
			t.Errorf("Clamp(%v,%v,%v) = %v", c[0], c[1], c[2], got)
		}
	}
}
