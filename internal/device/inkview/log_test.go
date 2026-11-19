package inkview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withLog(t *testing.T, verbose bool) string {
	t.Helper()
	oldLog, oldVerbose := DebugLog, Verbose
	t.Cleanup(func() { DebugLog, Verbose = oldLog, oldVerbose })
	DebugLog, Verbose = filepath.Join(t.TempDir(), "app.log"), verbose
	return DebugLog
}

func TestErrorsAreAlwaysLoggedAndTheTraceOnlyWhenVerbose(t *testing.T) {
	path := withLog(t, false)
	Logf("a trace line")
	Errorf("something went wrong: %d", 7)
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "trace") || !strings.Contains(string(got), "something went wrong: 7") {
		t.Errorf("a release logs errors only:\n%s", got)
	}

	Verbose = true
	Logf("now a trace line")
	logEvent(21, 0, 0)
	logEvent(31, 1, 2) // pointer moves are never logged
	got, _ = os.ReadFile(path)
	if !strings.Contains(string(got), "now a trace line") || !strings.Contains(string(got), "evt=21") || strings.Contains(string(got), "evt=31") {
		t.Errorf("verbose logs the trace:\n%s", got)
	}
}

func TestWithoutALogFileNothingIsWritten(t *testing.T) {
	withLog(t, true)
	DebugLog = ""
	Logf("x")
	Errorf("y")
	ResetLog() // nothing to reset
}

func TestALogThatGrewTooBigStartsAgain(t *testing.T) {
	path := withLog(t, false)
	Errorf("small")
	ResetLog()
	if got, _ := os.ReadFile(path); !strings.Contains(string(got), "small") {
		t.Error("a small log is kept")
	}
	os.WriteFile(path, []byte(strings.Repeat("x", MaxLogSize+1)), 0o644)
	ResetLog()
	if _, err := os.Stat(path); err == nil {
		t.Error("a log above MaxLogSize is removed")
	}
	Errorf("fresh")
	if got, _ := os.ReadFile(path); strings.Count(string(got), "\n") != 1 {
		t.Errorf("and starts again:\n%s", got)
	}
}

func TestAnUnwritableLogIsIgnored(t *testing.T) {
	withLog(t, true)
	DebugLog = filepath.Join(t.TempDir(), "no-such-dir", "app.log")
	Errorf("lost") // must not panic
}
