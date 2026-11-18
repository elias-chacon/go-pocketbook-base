package processlauncher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
)

// The adapter satisfies the domain's port.
var _ domain.AppLauncher = Launcher{}

func TestLaunchReplacesTheProgramAndTellsTheAppWhoStartedIt(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "hello.app")
	if err := os.WriteFile(exe, []byte("elf"), 0o755); err != nil {
		t.Fatal(err)
	}
	var gotPath string
	var gotEnv []string
	l := Launcher{Self: "/apps/launcher.app", Exec: func(path string, env []string) error {
		gotPath, gotEnv = path, env
		return nil
	}}
	if err := l.Launch(domain.Tile{ID: "hello", Exe: exe}); err != nil {
		t.Fatal(err)
	}
	if gotPath != exe {
		t.Errorf("started %q, want %q", gotPath, exe)
	}
	found := false
	for _, kv := range gotEnv {
		found = found || kv == platform.LauncherEnv+"=/apps/launcher.app"
	}
	if !found {
		t.Errorf("the app must be told which Launcher started it (%s): %v", platform.LauncherEnv, gotEnv)
	}
}

func TestLaunchFailsWhenTheFileIsGoneOrTheExecFails(t *testing.T) {
	called := false
	l := Launcher{Self: "x", Exec: func(string, []string) error { called = true; return nil }}
	err := l.Launch(domain.Tile{Exe: filepath.Join(t.TempDir(), "gone.app")})
	if err == nil || !strings.Contains(err.Error(), "missing") || called {
		t.Errorf("a deleted .app must fail without trying: err=%v called=%v", err, called)
	}

	exe := filepath.Join(t.TempDir(), "x.app")
	os.WriteFile(exe, []byte("x"), 0o755)
	boom := errors.New("exec format error")
	if err := (Launcher{Exec: func(string, []string) error { return boom }}).Launch(domain.Tile{Exe: exe}); err != boom {
		t.Errorf("an exec failure is returned as is: %v", err)
	}

	// with no Exec given the real one is used: it fails for a file that is not a program
	if err := (Launcher{}).Launch(domain.Tile{Exe: exe}); err == nil {
		t.Error("the default Exec must report a file that cannot be started")
	}
}
