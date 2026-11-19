package application

import (
	"errors"
	"testing"

	"go-pocketbook-base/internal/platform/domain"
)

type apps []domain.InstalledApp

func (a apps) Save(domain.InstalledApp) error { return nil }
func (a apps) List() []domain.InstalledApp    { return a }

func installedApp(id string, intents ...string) domain.InstalledApp {
	return domain.InstalledApp{
		Manifest: domain.Manifest{ID: id, Title: map[string]string{"pt-BR": id}, Description: map[string]string{"pt-BR": id}, Intents: intents},
		Exe:      "/apps/" + id + ".app",
	}
}

func TestSendOpensTheOtherAppWithTheIntentAndTheWayBack(t *testing.T) {
	var path, encoded string
	u := Intents{
		Apps:   apps{installedApp("mail"), installedApp("tasks", "tasks.create")},
		SelfID: "mail", SelfExe: "/apps/mail.app",
		Exec: func(p, e string) error { path, encoded = p, e; return nil },
	}
	if err := u.Send("tasks", "tasks.create", map[string]string{"mail": "42"}); err != nil {
		t.Fatal(err)
	}
	if path != "/apps/tasks.app" {
		t.Errorf("the target is started: %q", path)
	}
	in, err := domain.DecodeIntent(encoded)
	if err != nil || in.Action != "tasks.create" || in.From != "mail" || in.ReplyTo != "/apps/mail.app" || in.Data["mail"] != "42" {
		t.Errorf("%+v %v", in, err)
	}
}

func TestSendRefusesWhatCannotWorkWithoutLeavingTheApp(t *testing.T) {
	started := false
	u := Intents{
		Apps:   apps{installedApp("mail"), installedApp("tasks", "tasks.create")},
		SelfID: "mail", SelfExe: "/apps/mail.app",
		Exec: func(string, string) error { started = true; return nil },
	}
	cases := []struct {
		name, target, action string
		want                 error
	}{
		{"not installed", "ghost", "ghost.do", domain.ErrNoSuchApp},
		{"does not handle it", "tasks", "tasks.delete", domain.ErrNotHandled},
		{"itself", "mail", "mail.compose", domain.ErrNoSuchApp},
	}
	for _, c := range cases {
		if err := u.Send(c.target, c.action, nil); err != c.want {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	if err := u.Send("tasks", "bad", nil); err == nil {
		t.Error("a badly named action")
	}
	if err := u.Send("tasks", "tasks.create", map[string]string{"x": string(make([]byte, domain.MaxIntentBytes))}); err == nil {
		t.Error("an intent that is too big")
	}
	if started {
		t.Error("nothing was started")
	}

	boom := errors.New("exec format error")
	u.Exec = func(string, string) error { return boom }
	if err := u.Send("tasks", "tasks.create", nil); err != boom {
		t.Errorf("a failing exec is reported (the app stays): %v", err)
	}
}
