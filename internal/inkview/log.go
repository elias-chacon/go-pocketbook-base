package inkview

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// DebugLog, when non-empty, is a file path where every InkView event is
// appended (type, p1, p2). Set it from main to diagnose input problems on the
// device, e.g. inkview.DebugLog = "/mnt/ext1/applications/hello.log".
var DebugLog string

var logMu sync.Mutex

func logEvent(t, p1, p2 int) {
	// Skip noisy move/sync events (31 = pointer move, 39 = multitouch sync).
	if t == 31 || t == 39 {
		return
	}
	Logf("evt=%d p1=%d p2=%d", t, p1, p2)
}

// Logf appends a line to DebugLog (no-op when DebugLog is empty).
func Logf(format string, args ...interface{}) {
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
