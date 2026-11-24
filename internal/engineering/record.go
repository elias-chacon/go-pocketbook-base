package engineering

import (
	"bufio"
	"encoding/json"
	"io"
	"sort"
	"sync"
	"time"
)

// Recorder writes what the tests measure: one JSON object per line, with the time and the kind of
// event and the fields it was given. A nil Recorder records nothing. Every event reaches the file
// at once, except the touch, scroll and paint ones (see bulk), which wait for Flush.
type Recorder struct {
	mu  sync.Mutex
	w   *bufio.Writer
	now func() time.Time
}

// NewRecorder writes to w; now is the clock (time.Now when nil).
func NewRecorder(w io.Writer, now func() time.Time) *Recorder {
	if now == nil {
		now = time.Now
	}
	return &Recorder{w: bufio.NewWriter(w), now: now}
}

// Event records one event. The fields come in pairs, a name and a value:
// Event("draw", "mode", "a2", "ms", 12.5).
func (r *Recorder) Event(kind string, fields ...interface{}) {
	if r == nil {
		return
	}
	obj := map[string]interface{}{"t": r.now().Format(time.RFC3339Nano), "kind": kind}
	for i := 0; i+1 < len(fields); i += 2 {
		if name, ok := fields[i].(string); ok {
			obj[name] = fields[i+1]
		}
	}
	// Keys in a fixed order (t, kind, then the others sorted) so the file reads well.
	keys := make([]string, 0, len(obj))
	for k := range obj {
		if k != "t" && k != "kind" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	line := []byte(`{"t":` + quote(obj["t"]) + `,"kind":` + quote(kind))
	for _, k := range keys {
		v, err := json.Marshal(obj[k])
		if err != nil {
			continue
		}
		line = append(line, `,`+quote(k)+`:`...)
		line = append(line, v...)
	}
	line = append(line, '}', '\n')
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = r.w.Write(line)
	if !bulk[kind] {
		_ = r.w.Flush() // so that what was measured before a crash is in the file
	}
}

// bulk are the events that come many times a second while a finger moves: they are kept in the
// buffer (writing each one would slow the very thing being measured) until a test ends.
var bulk = map[string]bool{"touch": true, "scroll": true, "paint": true}

// Flush writes what is buffered to the file; call it when a test ends.
func (r *Recorder) Flush() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.w.Flush()
}

func quote(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
