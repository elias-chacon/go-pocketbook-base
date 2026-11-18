// Package processlauncher is the adapter that starts an app for the Launcher: it
// replaces the running Launcher with the app (execve), telling the app, through
// the environment, which Launcher started it so the app can come back.
package processlauncher

import (
	"fmt"
	"os"

	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/platform/infrastructure/process"
)

// Launcher is the domain.AppLauncher of the reader.
type Launcher struct {
	// Self is the path of the running Launcher: the app goes back to it.
	Self string
	// Exec replaces the process; nil uses process.Exec (tests replace it).
	Exec func(path string, env []string) error
}

// Launch implements domain.AppLauncher.
func (l Launcher) Launch(t domain.Tile) error {
	if _, err := os.Stat(t.Exe); err != nil {
		return fmt.Errorf("the app file is missing: %w", err)
	}
	exec := l.Exec
	if exec == nil {
		exec = process.Exec
	}
	return exec(t.Exe, process.WithEnv(platform.LauncherEnv, l.Self))
}
