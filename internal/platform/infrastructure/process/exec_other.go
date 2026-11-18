//go:build !linux

package process

// Exec is not available off Linux (the reader runs Linux; on the desktop the
// stub only reports it).
func Exec(path string, env []string) error { return ErrUnsupported }
