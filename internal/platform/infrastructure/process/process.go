// Package process starts one program in place of another: how the Launcher runs
// an app and how an app goes back to the Launcher. The reader has one screen and
// one InkView session, so the new program takes over the same process (execve)
// instead of running beside the old one.
package process

import (
	"errors"
	"os"
)

// ErrUnsupported is returned where Exec cannot work (anything but Linux).
var ErrUnsupported = errors.New("process: exec is only available on Linux (the reader)")

// WithoutEnv returns the current environment without key.
func WithoutEnv(key string) []string {
	prefix := key + "="
	var out []string
	for _, kv := range os.Environ() {
		if len(kv) < len(prefix) || kv[:len(prefix)] != prefix {
			out = append(out, kv)
		}
	}
	return out
}

// WithEnv returns the current environment plus key=value (replacing key).
func WithEnv(key, value string) []string {
	prefix := key + "="
	var out []string
	for _, kv := range os.Environ() {
		if len(kv) < len(prefix) || kv[:len(prefix)] != prefix {
			out = append(out, kv)
		}
	}
	return append(out, prefix+value)
}
