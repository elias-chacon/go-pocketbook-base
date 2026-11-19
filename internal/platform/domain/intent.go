package domain

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// IntentEnv is the environment variable through which an app opens another with
// something to do: its value is an encoded Intent. Like LauncherEnv it survives
// the execve that replaces one program with the next.
const IntentEnv = "PBGO_INTENT"

// MaxIntentBytes bounds the encoded intent: it travels in the environment.
const MaxIntentBytes = 48 << 10

// Intent is a request from one app to another: "create a task from this e-mail",
// "show this event". The apps never run side by side (the reader has one screen),
// so an intent is data handed over when one replaces the other.
type Intent struct {
	// Action names what is asked, "<app>.<verb>" such as "tasks.create". An app
	// declares in its manifest which actions it handles.
	Action string `json:"action"`
	// From is the id of the app that asked, ReplyTo the path of its executable:
	// when the app that was opened exits, the reader goes back to it.
	From    string `json:"from,omitempty"`
	ReplyTo string `json:"replyTo,omitempty"`
	// Data is what the action needs (an e-mail's id, a title...). Keep it small:
	// pass ids and let the target read the rest from the shared store.
	Data map[string]string `json:"data,omitempty"`
}

var actionPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}\.[a-z][a-z0-9-]{0,31}$`)

// The ways sending an intent fails.
var (
	// ErrNoSuchApp is returned when the app to open is not installed.
	ErrNoSuchApp = errors.New("platform: that app is not installed")
	// ErrNotHandled is returned when the app did not declare that it handles the action.
	ErrNotHandled = errors.New("platform: that app does not handle this action")
)

// ValidAction reports whether s is a well-formed action name, "app.verb".
func ValidAction(s string) bool { return actionPattern.MatchString(s) }

// Validate checks that the intent can be sent.
func (i Intent) Validate() error {
	if !ValidAction(i.Action) {
		return fmt.Errorf("platform: intent action %q must look like app.verb", i.Action)
	}
	if _, err := EncodeIntent(i); err != nil {
		return err
	}
	return nil
}

// EncodeIntent turns an intent into the text that goes in IntentEnv.
func EncodeIntent(i Intent) (string, error) {
	raw, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	text := base64.RawURLEncoding.EncodeToString(raw)
	if len(text) > MaxIntentBytes {
		return "", errors.New("platform: the intent is too big: pass ids, not content")
	}
	return text, nil
}

// DecodeIntent reads the text of IntentEnv. An empty text is no intent.
func DecodeIntent(text string) (*Intent, error) {
	if text == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("platform: the intent is not valid: %w", err)
	}
	var i Intent
	if err := json.Unmarshal(raw, &i); err != nil {
		return nil, fmt.Errorf("platform: the intent is not valid: %w", err)
	}
	if !ValidAction(i.Action) {
		return nil, fmt.Errorf("platform: the intent has a bad action %q", i.Action)
	}
	return &i, nil
}

// ReplyTarget decides where an app goes when it exits, now that it may have been
// opened by another app:
//
//   - reply is the ReplyTo of the intent it was opened with ("" when none);
//   - launcher, self and askedToExit are as in ReturnTarget.
//
// An app opened by another goes back to it; otherwise it follows ReturnTarget (back
// to the Launcher). Home always leaves: the system decides that, not the app.
func ReplyTarget(reply, launcher, self string, askedToExit bool) string {
	if askedToExit && reply != "" && reply != self {
		return reply
	}
	return ReturnTarget(launcher, self, askedToExit)
}
