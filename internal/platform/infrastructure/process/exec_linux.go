//go:build linux

package process

import "syscall"

// Exec replaces the running process with the program at path (execve): the new
// program takes over the same process, and a failure returns the error.
func Exec(path string, env []string) error {
	return syscall.Exec(path, []string{path}, env)
}
