package inkview

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// DebugLog, when non-empty, is the file path of the app's log, e.g.
// "/mnt/ext1/applications/hello.log". Errors are always written to it (see
// Errorf); the trace of every InkView event and of the app's steps (Logf) only
// when Verbose is true, so a release does not write a line per touch to the
// reader's storage.
var DebugLog string

// Verbose turns on the trace (Logf): every InkView event (type, p1, p2) and the
// app's steps. It is meant for diagnosing input or lifecycle problems.
var Verbose bool

// MaxLogSize is the size above which a log is started again (ResetLog).
const MaxLogSize = 64 << 10

var logMu sync.Mutex

func logEvent(t, p1, p2 int) {
	// Skip noisy move/sync events (31 = pointer move, 39 = multitouch sync).
	if t == 31 || t == 39 {
		return
	}
	Logf("evt=%d p1=%d p2=%d", t, p1, p2)
}

// Logf appends a line to the log when Verbose is on (a trace line).
func Logf(format string, args ...interface{}) {
	if Verbose {
		write(format, args...)
	}
}

// Errorf appends a line to the log whatever Verbose says: what went wrong
// (a failed download, a failed registration) must be readable after the fact.
// Never pass it a password, a token or the text of a message.
func Errorf(format string, args ...interface{}) { write(format, args...) }

// ResetLog empties the log when it has grown beyond MaxLogSize, so it does not
// fill the reader over months of use. Call it once when the app starts.
func ResetLog() {
	if DebugLog == "" {
		return
	}
	logMu.Lock()
	defer logMu.Unlock()
	if st, err := os.Stat(DebugLog); err == nil && st.Size() > MaxLogSize {
		os.Remove(DebugLog)
	}
}

func write(format string, args ...interface{}) {
	if DebugLog == "" {
		return
	}
	logMu.Lock()
	defer logMu.Unlock()
	f, err := os.OpenFile(DebugLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}
