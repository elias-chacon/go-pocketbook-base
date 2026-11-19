package domain

import (
	"strings"
	"testing"
)

func TestAnIntentSurvivesTheEnvironment(t *testing.T) {
	in := Intent{Action: "tasks.create", From: "mail", ReplyTo: "/mnt/ext1/applications/mail.app", Data: map[string]string{"mail": "42", "title": "Pagar o aluguel"}}
	text, err := EncodeIntent(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(text, "\x00=\n ") {
		t.Errorf("fit for an environment variable: %q", text)
	}
	got, err := DecodeIntent(text)
	if err != nil || got == nil || got.Action != "tasks.create" || got.From != "mail" || got.Data["title"] != "Pagar o aluguel" || got.ReplyTo != in.ReplyTo {
		t.Errorf("%+v %v", got, err)
	}
	if none, err := DecodeIntent(""); none != nil || err != nil {
		t.Errorf("no intent: %v %v", none, err)
	}
}

func TestBrokenIntentsAreRefused(t *testing.T) {
	for name, text := range map[string]string{
		"not base64": "!!!",
		"not json":   "aGVsbG8",                 // hello
		"bad action": "eyJhY3Rpb24iOiJub3BlIn0", // {"action":"nope"}
		"no action":  "e30",                     // {}
	} {
		if in, err := DecodeIntent(text); err == nil || in != nil {
			t.Errorf("%s must be refused: %v %v", name, in, err)
		}
	}
}

func TestActionsLookLikeAppDotVerb(t *testing.T) {
	for _, ok := range []string{"tasks.create", "calendar.show-event", "mail.compose"} {
		if !ValidAction(ok) {
			t.Errorf("%q is valid", ok)
		}
	}
	for _, bad := range []string{"", "tasks", "Tasks.create", "tasks.", ".create", "a.b.c", "1tasks.create", "tasks.create now", strings.Repeat("a", 40) + ".x"} {
		if ValidAction(bad) {
			t.Errorf("%q is not valid", bad)
		}
	}
	if err := (Intent{Action: "nope"}).Validate(); err == nil {
		t.Error("an intent needs an action")
	}
}

func TestAHugeIntentIsRefused(t *testing.T) {
	in := Intent{Action: "mail.compose", Data: map[string]string{"body": strings.Repeat("x", MaxIntentBytes)}}
	if _, err := EncodeIntent(in); err == nil || !strings.Contains(err.Error(), "too big") {
		t.Errorf("pass ids, not content: %v", err)
	}
	if err := in.Validate(); err == nil {
		t.Error("Validate sees it too")
	}
}

func TestAnAppGoesBackToWhoOpenedItOrElseToTheLauncher(t *testing.T) {
	cases := []struct {
		name                  string
		reply, launcher, self string
		askedToExit           bool
		want                  string
	}{
		{"opened by another app", "/a/mail.app", "/a/launcher.app", "/a/tasks.app", true, "/a/mail.app"},
		{"opened by another app, no launcher", "/a/mail.app", "", "/a/tasks.app", true, "/a/mail.app"},
		{"Home always leaves", "/a/mail.app", "/a/launcher.app", "/a/tasks.app", false, ""},
		{"no intent: the launcher", "", "/a/launcher.app", "/a/tasks.app", true, "/a/launcher.app"},
		{"nobody to go back to", "", "", "/a/tasks.app", true, ""},
		{"an app does not reply to itself", "/a/tasks.app", "/a/launcher.app", "/a/tasks.app", true, "/a/launcher.app"},
	}
	for _, c := range cases {
		if got := ReplyTarget(c.reply, c.launcher, c.self, c.askedToExit); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestAManifestDeclaresTheIntentsItHandles(t *testing.T) {
	m := Manifest{ID: "tasks", Title: map[string]string{"pt-BR": "Tarefas"}, Description: map[string]string{"pt-BR": "d"}, Intents: []string{"tasks.create", "tasks.show"}}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if !m.Handles("tasks.create") || m.Handles("tasks.delete") || (Manifest{}).Handles("tasks.create") {
		t.Error("Handles")
	}
	m.Intents = []string{"create"}
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "app.verb") {
		t.Errorf("a badly named action: %v", err)
	}
}
