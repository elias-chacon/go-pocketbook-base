package application

import (
	"go-pocketbook-base/internal/launcher/domain"
)

// The phases a startup tells its Progress it is in (the screen turns them into text).
const (
	PhaseHomeScreen = "home-screen" // looking at the home screen
	PhaseRemoving   = "removing"    // taking an entry out of it (the detail is the id)
	PhaseCatalog    = "catalog"     // bringing the catalog
)

// The names of the startup tasks, to read their result in an Outcome.
const (
	TaskHomeScreen = "home-screen"
	TaskCatalog    = "catalog"
)

// Context is what the tasks of a startup share: what they are given and what they find.
type Context struct {
	Lang     string
	Progress *domain.Progress // the startup began it with the weight of every task: a task steps it as it works
	Listings []domain.Listing // set by the catalog task
	Removed  []string         // set by the home screen task: the entries it took out
}

// Task is one thing the Launcher does when it starts.
type Task struct {
	Name string
	// Weight is how many steps the task gives the Progress (it steps them itself; a task that
	// learns more on the way adds them).
	Weight int
	Run    func(c *Context) error
}

// Startup is the group of tasks the Launcher runs when it opens, in order and in the background.
// Every task is tried even when an earlier one failed: a failure is a result, not the end.
type Startup struct{ Tasks []Task }

// Outcome is what a startup left behind.
type Outcome struct {
	Context
	Errors map[string]error // by task name: only the ones that failed
}

// Execute runs the tasks (blocking: the screen calls it from a goroutine) and reports how far it
// got through p (nil: no progress).
func (s Startup) Execute(lang string, p *domain.Progress) Outcome {
	total := 0
	for _, t := range s.Tasks {
		total += t.Weight
	}
	p.Begin(total)
	defer p.End()
	out := Outcome{Context: Context{Lang: lang, Progress: p}, Errors: map[string]error{}}
	for _, t := range s.Tasks {
		if err := t.Run(&out.Context); err != nil {
			out.Errors[t.Name] = err
		}
	}
	return out
}

// FetchCatalog is the task that brings the latest catalog of the apps (2 steps: the release and
// its catalog.json; the catalog adds one per icon it fetches).
func FetchCatalog(cat domain.Catalog) Task {
	return Task{Name: TaskCatalog, Weight: 2, Run: func(c *Context) error {
		c.Progress.SetPhase(PhaseCatalog, "")
		list, err := cat.List(c.Lang)
		c.Listings = list
		return err
	}}
}

// PruneHomeScreen is the task that takes out of the home screen the entries of apps that are no
// longer on the reader (1 step to look, one more for each entry it takes out, told to the Progress
// as it goes).
func PruneHomeScreen(h domain.HomeScreen) Task {
	return Task{Name: TaskHomeScreen, Weight: 1, Run: func(c *Context) error {
		c.Progress.SetPhase(PhaseHomeScreen, "")
		removed, err := h.RemoveOrphans(func(id string) {
			c.Progress.Add(1)
			c.Progress.SetPhase(PhaseRemoving, id)
			c.Progress.Step()
		})
		c.Removed = removed
		c.Progress.Step()
		return err
	}}
}
