package application

import (
	"go-pocketbook-base/internal/platform/domain"
)

// Intents is the use case "open another app with something to do": the app
// replaces itself with the other (the reader runs one app at a time) and hands it
// an Intent; when the other exits, the reader comes back to this one.
type Intents struct {
	Apps    domain.AppRegistry
	SelfID  string // the id of the running app
	SelfExe string // the path of its executable: where the other app returns to
	// Exec replaces the running program with the one at path, giving it the
	// encoded intent (the composition root puts it in the environment). It does
	// not return when it works.
	Exec func(path, encodedIntent string) error
}

// Send opens app targetID with an action. It fails, without leaving this app,
// when the app is not installed or did not declare that it handles the action.
func (u Intents) Send(targetID, action string, data map[string]string) error {
	in := domain.Intent{Action: action, From: u.SelfID, ReplyTo: u.SelfExe, Data: data}
	if err := in.Validate(); err != nil {
		return err
	}
	target, ok := u.find(targetID)
	if !ok {
		return domain.ErrNoSuchApp
	}
	if !target.Handles(action) {
		return domain.ErrNotHandled
	}
	text, err := domain.EncodeIntent(in)
	if err != nil {
		return err
	}
	return u.Exec(target.Exe, text)
}

func (u Intents) find(id string) (domain.InstalledApp, bool) {
	if id == u.SelfID {
		return domain.InstalledApp{}, false // an app does not send itself an intent
	}
	for _, a := range u.Apps.List() {
		if a.ID == id {
			return a, true
		}
	}
	return domain.InstalledApp{}, false
}
