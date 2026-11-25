package domain

import "testing"

func TestProgressCountsStepsAndGrowsWhenTheJobIsLearned(t *testing.T) {
	var p Progress
	if f, on := p.State(); f != 0 || on {
		t.Fatalf("a new progress is idle: %v %v", f, on)
	}
	p.Begin(2)
	p.Step()
	if f, on := p.State(); f != 0.5 || !on {
		t.Errorf("1 of 2 = %v %v", f, on)
	}
	p.Step()
	p.Add(2) // two icons to fetch
	if f, _ := p.State(); f != 0.5 {
		t.Errorf("2 of 4 = %v", f)
	}
	p.End()
	if f, on := p.State(); f != 1 || on {
		t.Errorf("an ended job is complete and idle: %v %v", f, on)
	}
}

func TestProgressNeverPassesItsTotalAndIsSafeWhenNil(t *testing.T) {
	var p Progress
	p.Begin(1)
	p.Step()
	p.Step()
	if f, _ := p.State(); f != 1 {
		t.Errorf("fraction = %v", f)
	}
	var none *Progress
	none.Begin(3)
	none.Add(1)
	none.Step()
	none.End()
	if f, on := none.State(); f != 0 || on {
		t.Errorf("nil = %v %v", f, on)
	}
}
