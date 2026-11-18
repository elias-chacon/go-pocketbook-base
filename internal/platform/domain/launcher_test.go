package domain

import "testing"

func TestAnAppGoesBackToTheLauncherOnlyWhenItWasStartedByOneAndAskedToExit(t *testing.T) {
	const launcher, self = "/mnt/ext1/applications/launcher.app", "/mnt/ext1/applications/hello.app"
	cases := []struct {
		name        string
		launcher    string
		self        string
		askedToExit bool
		want        string
	}{
		{"started by the launcher, asks to exit", launcher, self, true, launcher},
		{"started by the launcher, Home pressed (the system decides)", launcher, self, false, ""},
		{"started from the home screen", "", self, true, ""},
		{"the launcher itself inherited the variable: never return to itself", launcher, launcher, true, ""},
	}
	for _, c := range cases {
		if got := ReturnTarget(c.launcher, c.self, c.askedToExit); got != c.want {
			t.Errorf("%s: ReturnTarget = %q, want %q", c.name, got, c.want)
		}
	}
}
