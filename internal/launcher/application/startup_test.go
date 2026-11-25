package application

import (
	"errors"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
)

type stubCatalog struct {
	list []domain.Listing
	err  error
	lang string
}

func (c *stubCatalog) List(lang string) ([]domain.Listing, error) {
	c.lang = lang
	return c.list, c.err
}

type stubHome struct {
	removed []string
	err     error
}

func (h stubHome) RemoveOrphans(report func(string)) ([]string, error) {
	for _, id := range h.removed {
		report(id)
	}
	return h.removed, h.err
}

func TestStartupRunsEveryTaskInOrderAndGathersWhatTheyFound(t *testing.T) {
	cat := &stubCatalog{list: []domain.Listing{{ID: "a"}}}
	var p domain.Progress
	out := Startup{Tasks: []Task{PruneHomeScreen(stubHome{removed: []string{"old"}}), FetchCatalog(cat)}}.Execute("en", &p)
	if len(out.Errors) != 0 || len(out.Removed) != 1 || out.Removed[0] != "old" || len(out.Listings) != 1 || cat.lang != "en" {
		t.Errorf("outcome = %+v (lang %q)", out, cat.lang)
	}
	if f, on := p.State(); f != 1 || on {
		t.Errorf("an ended startup is complete and idle: %v %v", f, on)
	}
}

func TestAFailedTaskDoesNotStopTheOthers(t *testing.T) {
	boom := errors.New("boom")
	cat := &stubCatalog{list: []domain.Listing{{ID: "a"}}}
	out := Startup{Tasks: []Task{PruneHomeScreen(stubHome{err: boom}), FetchCatalog(cat)}}.Execute("pt-BR", nil)
	if out.Errors[TaskHomeScreen] != boom || len(out.Errors) != 1 || len(out.Listings) != 1 {
		t.Errorf("the catalog still comes when the home screen fails: %+v", out)
	}
	out = Startup{Tasks: []Task{FetchCatalog(&stubCatalog{err: boom})}}.Execute("pt-BR", nil)
	if out.Errors[TaskCatalog] != boom {
		t.Errorf("the catalog failure is a result: %+v", out.Errors)
	}
}

func TestTheProgressStartsWithTheWeightOfEveryTask(t *testing.T) {
	var p domain.Progress
	seen := 0.0
	Startup{Tasks: []Task{
		{Name: "a", Weight: 1, Run: func(c *Context) error { c.Progress.Step(); seen, _ = c.Progress.State(); return nil }},
		{Name: "b", Weight: 3, Run: func(c *Context) error { return nil }},
	}}.Execute("en", &p)
	if seen != 0.25 {
		t.Errorf("1 of 4 steps = %v", seen)
	}
}

func TestTheProgressIsToldEachEntryAsItIsTakenOutAndThenTheCatalog(t *testing.T) {
	var p domain.Progress
	var seen []string
	check := Task{Name: "spy", Run: func(c *Context) error {
		phase, _ := c.Progress.Phase()
		seen = append(seen, phase)
		return nil
	}}
	cat := FetchCatalog(&stubCatalog{})
	spyCatalog := Task{Name: TaskCatalog, Weight: 2, Run: func(c *Context) error {
		err := cat.Run(c)
		phase, _ := c.Progress.Phase()
		seen = append(seen, phase)
		return err
	}}
	prune := PruneHomeScreen(stubHome{removed: []string{"a", "b"}})
	spyPrune := Task{Name: TaskHomeScreen, Weight: 1, Run: func(c *Context) error {
		err := prune.Run(c)
		phase, detail := c.Progress.Phase()
		seen = append(seen, phase+":"+detail)
		return err
	}}
	Startup{Tasks: []Task{spyPrune, check, spyCatalog}}.Execute("en", &p)
	want := []string{PhaseRemoving + ":b", PhaseRemoving, PhaseCatalog}
	if len(seen) != 3 || seen[0] != want[0] || seen[1] != want[1] || seen[2] != want[2] {
		t.Errorf("phases seen = %v, want %v", seen, want)
	}
}
