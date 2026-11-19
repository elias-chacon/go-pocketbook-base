package platform

import (
	"errors"
	"testing"
)

func TestReturnToLauncherStartsTheLauncherOnlyWhenItShould(t *testing.T) {
	var started []string
	exec := func(path string, env []string) error { started = append(started, path); return nil }

	returnToLauncher("/a/hello.app", true, "/a/launcher.app", "", exec)
	if len(started) != 1 || started[0] != "/a/launcher.app" {
		t.Fatalf("an app that asked to exit goes back: %v", started)
	}

	started = nil
	returnToLauncher("/a/hello.app", false, "/a/launcher.app", "", exec)   // Home
	returnToLauncher("/a/hello.app", true, "", "", exec)                   // not started by a launcher
	returnToLauncher("/a/launcher.app", true, "/a/launcher.app", "", exec) // the launcher itself
	if len(started) != 0 {
		t.Errorf("nothing else may start anything: %v", started)
	}
}

func TestReturnToLauncherSurvivesAFailedExec(t *testing.T) {
	failing := func(string, []string) error { return errors.New("exec format error") }
	returnToLauncher("/a/hello.app", true, "/a/launcher.app", "", failing) // only logged, never a panic
}

func TestAnAppOpenedByAnotherGoesBackToItWithoutTheIntent(t *testing.T) {
	var started []string
	var env []string
	exec := func(path string, e []string) error { started = append(started, path); env = e; return nil }
	t.Setenv("PBGO_INTENT", "stale-intent")
	t.Setenv("PBGO_KEEP", "yes")

	returnToLauncher("/a/tasks.app", true, "/a/launcher.app", "/a/mail.app", exec)
	if len(started) != 1 || started[0] != "/a/mail.app" {
		t.Fatalf("back to the app that asked: %v", started)
	}
	kept := false
	for _, kv := range env {
		if kv == "PBGO_INTENT=stale-intent" {
			t.Error("the app it returns to must not find the intent it sent")
		}
		kept = kept || kv == "PBGO_KEEP=yes"
	}
	if !kept {
		t.Error("the rest of the environment (PBGO_LAUNCHER...) is kept")
	}

	started = nil
	returnToLauncher("/a/tasks.app", false, "/a/launcher.app", "/a/mail.app", exec) // Home
	if len(started) != 0 {
		t.Errorf("Home leaves: %v", started)
	}
}
