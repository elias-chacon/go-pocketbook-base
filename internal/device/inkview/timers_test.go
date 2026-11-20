package inkview

import (
	"testing"
	"time"
)

func TestTimersRunWhatIsDueInOrderAndKeepTheRest(t *testing.T) {
	var q timers
	now := time.Now()
	var ran []string
	if d := q.add(now, 50*time.Millisecond, func() { ran = append(ran, "a") }); d != 50*time.Millisecond {
		t.Errorf("first entry due in %v", d)
	}
	if d := q.add(now, 10*time.Millisecond, func() { ran = append(ran, "b") }); d != 10*time.Millisecond {
		t.Errorf("the earliest is now b: %v", d)
	}
	q.add(now, 5*time.Second, func() { ran = append(ran, "late") })
	next := q.runDue(now.Add(time.Second))
	if len(ran) != 2 || ran[0] != "a" || ran[1] != "b" {
		t.Errorf("ran %v", ran)
	}
	if next < 3*time.Second {
		t.Errorf("the late one is still waiting: %v", next)
	}
	if q.runDue(now.Add(10*time.Second)) != -1 || len(ran) != 3 {
		t.Errorf("the last one ran: %v", ran)
	}
}

func TestAnEntryAddedWhileRunningWaitsForTheNextRound(t *testing.T) {
	var q timers
	now := time.Now()
	count := 0
	var again func()
	again = func() {
		count++
		if count < 5 {
			q.add(time.Now(), 0, again)
		}
	}
	q.add(now, 0, again)
	q.runDue(now.Add(time.Second))
	if count != 1 {
		t.Errorf("one run per round, got %d", count)
	}
}
