package domain

// LauncherEnv is the environment variable through which the Launcher tells an app
// it started: its value is the path of the Launcher's own executable. An app
// started this way goes back to the Launcher when it asks to exit (the Exit menu,
// the Back key); an app started from the home screen simply leaves. Pressing
// Home always leaves: that is the system's, not the app's, decision.
const LauncherEnv = "PBGO_LAUNCHER"

// ReturnTarget decides where an app goes when it exits.
//
//   - launcher is the value of LauncherEnv ("" when the app was not started by one);
//   - self is the path of the running app: the Launcher itself inherits the
//     variable from the app that returned to it, and must not return to itself;
//   - askedToExit is true when the app ended on its own accord (not by Home).
//
// It returns the executable to start, or "" when the app should just end.
func ReturnTarget(launcher, self string, askedToExit bool) string {
	if !askedToExit || launcher == "" || launcher == self {
		return ""
	}
	return launcher
}
