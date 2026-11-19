package domain

import (
	"strings"
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

var today = day(2026, 3, 10) // a Tuesday

func TestATaskNeedsATitleAndARepeatingOneNeedsADueDate(t *testing.T) {
	if err := (Task{Title: "Pagar o aluguel"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if (Task{Title: " "}).Validate() == nil {
		t.Error("a title is required")
	}
	err := (Task{Title: "x", Recurrence: "FREQ=WEEKLY"}).Validate()
	if err == nil || !strings.Contains(err.Error(), "due date") {
		t.Errorf("%v", err)
	}
	if err := (Task{Title: "x", Due: today, Recurrence: "FREQ=SOMETIMES"}).Validate(); err == nil || !strings.Contains(err.Error(), "recurrence") {
		t.Errorf("%v", err)
	}
	if (Task{Title: "x", Due: today, Recurrence: "FREQ=DAILY"}).Validate() != nil {
		t.Error("a good rule")
	}
}

func TestDayIsMidnightUTCOfTheCalendarDay(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	got := Day(time.Date(2026, 3, 10, 23, 30, 0, 0, ny))
	if !got.Equal(day(2026, 3, 10)) || got.Location() != time.UTC {
		t.Errorf("%v", got)
	}
}

func TestOverdueMeansOpenAndDueBeforeToday(t *testing.T) {
	cases := map[string]struct {
		task Task
		want bool
	}{
		"due yesterday":     {Task{Due: day(2026, 3, 9)}, true},
		"due today":         {Task{Due: today}, false},
		"due tomorrow":      {Task{Due: day(2026, 3, 11)}, false},
		"no due date":       {Task{}, false},
		"done and past due": {Task{Due: day(2026, 3, 9), Done: true}, false},
	}
	for name, c := range cases {
		if got := c.task.Overdue(today); got != c.want {
			t.Errorf("%s: %v", name, got)
		}
	}
}

func TestStepsAreCounted(t *testing.T) {
	task := Task{Steps: []Step{{"a", true}, {"b", false}, {"c", true}}}
	if task.StepsDone() != 2 || task.StepsTotal() != 3 {
		t.Errorf("%d of %d", task.StepsDone(), task.StepsTotal())
	}
}

func TestCompletingASimpleTaskTicksItOnce(t *testing.T) {
	now := time.Date(2026, 3, 10, 15, 0, 0, 0, time.UTC)
	done, next, err := (Task{Title: "x"}).Complete(now)
	if err != nil || !done.Done || !done.DoneAt.Equal(now) || next != nil {
		t.Errorf("%+v %v %v", done, next, err)
	}
	again, next, _ := done.Complete(now.Add(time.Hour))
	if !again.DoneAt.Equal(now) || next != nil {
		t.Error("ticking a done task changes nothing")
	}
}

func TestCompletingARepeatingTaskGivesTheNextOne(t *testing.T) {
	weekly := Task{
		ID: "t1", RemoteID: "r1", Title: "Relatório", Due: day(2026, 3, 6), // a Friday
		Recurrence: "FREQ=WEEKLY;BYDAY=FR", Important: true, MyDay: today, Reminder: today,
		Steps: []Step{{"rascunho", true}, {"revisão", true}},
	}
	// done on time (the 6th): the next Friday
	done, next, err := weekly.Complete(day(2026, 3, 6))
	if err != nil || !done.Done || next == nil {
		t.Fatalf("%v %v", next, err)
	}
	if !next.Due.Equal(day(2026, 3, 13)) || next.Done || next.ID != "" || next.RemoteID != "" || !next.Important {
		t.Errorf("%+v", next)
	}
	if !next.MyDay.IsZero() || !next.Reminder.IsZero() || len(next.Steps) != 2 || next.Steps[0].Done || next.Steps[0].Title != "rascunho" {
		t.Errorf("the next one starts clean: %+v", next)
	}

	// done a month late: the next Friday after today, not a pile of overdue copies
	_, late, _ := weekly.Complete(day(2026, 4, 8))
	if late == nil || !late.Due.Equal(day(2026, 4, 10)) {
		t.Errorf("%+v", late)
	}
}

func TestARepeatingTaskWhoseRuleEndedGivesNoNext(t *testing.T) {
	once := Task{Title: "x", Due: day(2026, 3, 6), Recurrence: "FREQ=DAILY;COUNT=1"}
	if _, next, err := once.Complete(day(2026, 3, 6)); err != nil || next != nil {
		t.Errorf("%v %v", next, err)
	}
	bad := Task{Title: "x", Recurrence: "FREQ=DAILY"} // no due date
	if _, _, err := bad.Complete(today); err == nil {
		t.Error("a repeating task without a due date cannot repeat")
	}
}

func sample() []Task {
	return []Task{
		{ID: "a", Title: "A", CreatedAt: day(2026, 3, 1), MyDay: today},
		{ID: "b", Title: "B", CreatedAt: day(2026, 3, 2), Important: true, Due: day(2026, 3, 20)},
		{ID: "c", Title: "C", CreatedAt: day(2026, 3, 3), Due: day(2026, 3, 8), MyDay: today},
		{ID: "d", Title: "D", CreatedAt: day(2026, 3, 4), Important: true, Done: true, Due: day(2026, 3, 1), MyDay: today},
		{ID: "e", Title: "E", CreatedAt: day(2026, 3, 5), Due: day(2026, 3, 12), Important: true, MyDay: day(2026, 3, 9)},
	}
}

func ids(tasks []Task) string {
	var s []string
	for _, t := range tasks {
		s = append(s, t.ID)
	}
	return strings.Join(s, "")
}

func TestViewsFilterAndOrderTheTasks(t *testing.T) {
	cases := map[View]string{
		ViewMyDay:     "ca",  // important first (none), then newest first: c (3rd) before a (1st)
		ViewImportant: "eb",  // newest first among the important
		ViewPlanned:   "ceb", // by due date: c (8th), e (12th), b (20th)
		ViewOverdue:   "c",
		ViewAll:       "edbca", // the important ones first, newest first (e, d, b), then the others (c, a)
	}
	for v, want := range cases {
		if got := ids(Filter(sample(), v, today)); got != want {
			t.Errorf("%s: %q, want %q", v, got, want)
		}
	}
	if len(Filter(sample(), ViewAll, today)) != 5 {
		t.Error("the all view keeps the done ones")
	}
	if len(Filter(sample(), View("nonsense"), today)) != 0 {
		t.Error("an unknown view is empty")
	}
}

func TestCountsAreTheOpenTasksOfEachView(t *testing.T) {
	got := Count(sample(), today)
	if got != (Counts{MyDay: 2, Important: 2, Planned: 3, Overdue: 1}) {
		t.Errorf("%+v", got)
	}
}
