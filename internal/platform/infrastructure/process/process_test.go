package process

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestWithEnvAddsOrReplacesAVariableAndKeepsTheRest(t *testing.T) {
	t.Setenv("PBGO_TEST_KEEP", "yes")
	t.Setenv("PBGO_TEST_VAR", "old")

	env := WithEnv("PBGO_TEST_VAR", "new")
	count, kept := 0, false
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, "PBGO_TEST_VAR="):
			count++
			if kv != "PBGO_TEST_VAR=new" {
				t.Errorf("the value was not replaced: %s", kv)
			}
		case kv == "PBGO_TEST_KEEP=yes":
			kept = true
		}
	}
	if count != 1 || !kept {
		t.Errorf("exactly one PBGO_TEST_VAR and the rest kept: count=%d kept=%v", count, kept)
	}
	if got := os.Getenv("PBGO_TEST_VAR"); got != "old" {
		t.Errorf("our own environment must not change: %q", got)
	}
	if env := WithEnv("PBGO_TEST_BRAND_NEW", "1"); env[len(env)-1] != "PBGO_TEST_BRAND_NEW=1" {
		t.Error("a new variable is appended")
	}
}

func TestExecReportsFailureInsteadOfPanicking(t *testing.T) {
	err := Exec("/nonexistent/program", nil)
	if err == nil {
		t.Fatal("starting a program that does not exist must fail")
	}
	if runtime.GOOS != "linux" && err != ErrUnsupported {
		t.Errorf("off Linux the error is ErrUnsupported: %v", err)
	}
}

func TestWithoutEnvRemovesOnlyThatVariable(t *testing.T) {
	t.Setenv("PBGO_TEST_GONE", "x")
	t.Setenv("PBGO_TEST_STAYS", "y")
	gone, stays := false, false
	for _, kv := range WithoutEnv("PBGO_TEST_GONE") {
		gone = gone || strings.HasPrefix(kv, "PBGO_TEST_GONE=")
		stays = stays || kv == "PBGO_TEST_STAYS=y"
	}
	if gone || !stays {
		t.Errorf("gone=%v stays=%v", gone, stays)
	}
}
