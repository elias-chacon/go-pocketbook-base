package platform

import (
	"errors"
	"testing"
)

func TestReturnToLauncherStartsTheLauncherOnlyWhenItShould(t *testing.T) {
	var started []string
	exec := func(path string, env []string) error { started = append(started, path); return nil }

	returnToLauncher("/a/hello.app", true, "/a/launcher.app", exec)
	if len(started) != 1 || started[0] != "/a/launcher.app" {
		t.Fatalf("an app that asked to exit goes back: %v", started)
	}

	started = nil
	returnToLauncher("/a/hello.app", false, "/a/launcher.app", exec)   // Home
	returnToLauncher("/a/hello.app", true, "", exec)                   // not started by a launcher
	returnToLauncher("/a/launcher.app", true, "/a/launcher.app", exec) // the launcher itself
	if len(started) != 0 {
		t.Errorf("nothing else may start anything: %v", started)
	}
}

func TestReturnToLauncherSurvivesAFailedExec(t *testing.T) {
	failing := func(string, []string) error { return errors.New("exec format error") }
	returnToLauncher("/a/hello.app", true, "/a/launcher.app", failing) // only logged, never a panic
}
