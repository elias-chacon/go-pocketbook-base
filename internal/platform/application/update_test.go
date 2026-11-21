package application

import (
	"errors"
	"testing"
	"time"

	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type fakeSource struct {
	update domain.Update
	err    error
	asked  int
}

func (f *fakeSource) Latest(string) (domain.Update, error) { f.asked++; return f.update, f.err }

type fakeInstaller struct {
	got domain.Update
	err error
}

func (f *fakeInstaller) Install(_ string, u domain.Update) error { f.got = u; return f.err }

type memState struct{ st domain.UpdateState }

func (m *memState) Load() domain.UpdateState        { return m.st }
func (m *memState) Save(s domain.UpdateState) error { m.st = s; return nil }

func newUpdater(current string, src *fakeSource) (Updater, *fakeInstaller, *memState, *int) {
	inst, st, restarts := &fakeInstaller{}, &memState{}, new(int)
	return Updater{
		ID: "hello", Current: current, Source: src, Install: inst, State: st,
		Installed: func() (string, error) { return "aaa", nil },
		Restart:   func() error { *restarts++; return nil },
		Now:       func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) },
	}, inst, st, restarts
}

func TestCheckOffersOnlyANewerReleaseAndNeverToADevBuild(t *testing.T) {
	src := &fakeSource{update: domain.Update{Version: "1.50.0", URL: "u", AppSHA256: "bbb"}}
	u, _, _, _ := newUpdater("1.49.3", src)
	if up, ok := u.Check(); !ok || up.Version != "1.50.0" {
		t.Fatalf("a newer release must be offered: %+v %v", up, ok)
	}
	src2 := &fakeSource{update: domain.Update{Version: "9.9.9", AppSHA256: "AAA"}}
	same, _, _, _ := newUpdater("1.50.0", src2)
	if _, ok := same.Check(); ok {
		t.Error("the installed checksum is not an update, whatever the version says")
	}
	dev, _, _, _ := newUpdater("dev", src)
	asked := src.asked
	if _, ok := dev.Check(); ok || src.asked != asked {
		t.Error("a developer build is never replaced, and does not even ask the network")
	}
}

func TestCheckOfflineAnswersNoAndTriesAgainNextTime(t *testing.T) {
	src := &fakeSource{err: errors.New("no route to host")}
	u, _, st, _ := newUpdater("1.0.0", src)
	if _, ok := u.Check(); ok {
		t.Fatal("offline must offer nothing")
	}
	if !st.st.CheckedAt.IsZero() {
		t.Error("a failed check must not count as a check")
	}
}

func TestACheckKeepsTheNetworkQuietForAWhileButStillOffersWhatItFound(t *testing.T) {
	src := &fakeSource{update: domain.Update{Version: "2.0.0", AppSHA256: "ccc"}}
	u, _, _, _ := newUpdater("1.0.0", src)
	u.Check()
	if up, ok := u.Check(); !ok || up.Version != "2.0.0" || src.asked != 1 {
		t.Errorf("the second opening reuses what the first found: %+v %v, asked %d times", up, ok, src.asked)
	}
}

func TestPostponingAVersionSilencesItButNotTheNextOne(t *testing.T) {
	src := &fakeSource{update: domain.Update{Version: "1.1.0", AppSHA256: "d1"}}
	u, _, st, _ := newUpdater("1.0.0", src)
	up, _ := u.Check()
	u.Postpone(up)
	if _, ok := u.Check(); ok {
		t.Error("a postponed version is not offered again")
	}
	st.st.CheckedAt = time.Time{} // time passes
	src.update = domain.Update{Version: "1.2.0", AppSHA256: "d2"}
	if up, ok := u.Check(); !ok || up.Version != "1.2.0" {
		t.Errorf("a newer one is offered: %+v %v", up, ok)
	}
}

func TestApplyInstallsThenRestartsAndStopsOnFailure(t *testing.T) {
	u, inst, _, restarts := newUpdater("1.0.0", &fakeSource{})
	if err := u.Apply(domain.Update{Version: "1.1.0"}); err != nil || inst.got.Version != "1.1.0" || *restarts != 1 {
		t.Fatalf("apply: %v %+v restarts %d", err, inst.got, *restarts)
	}
	inst.err = errors.New("disk full")
	if err := u.Apply(domain.Update{}); err == nil || *restarts != 1 {
		t.Errorf("a failed install must not restart: %v restarts %d", err, *restarts)
	}
}

// ---- in the runtime ----

func startWithUpdates(t *testing.T, src *fakeSource, lang string) (*Runtime, *recorder, *fakeInstaller, *int, *fakeApp) {
	t.Helper()
	doc := widget.NewDocument(1404, 1872)
	doc.SetRoot(widget.NewColumn(0, widget.NewButton("x")))
	app := &fakeApp{doc: doc}
	rec := &recorder{}
	u, inst, _, restarts := newUpdater("1.0.0", src)
	rt := NewRuntime(Env{Lang: lang, Version: "1.0.0"}, func(Env) App { return app }, rec).WithUpdates(u)
	rt.Start(1404, 1872, &widget.FakeKeyboard{})
	return rt, rec, inst, restarts, app
}

// settle runs the scheduled polls until the background check answered and the dialog is up.
func settle(t *testing.T, rec *recorder, app *fakeApp) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for app.doc.Modal() == nil && time.Now().Before(deadline) {
		later := rec.later
		rec.later = nil
		for _, f := range later {
			f()
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestTheRuntimeOffersAnUpdateInTheBackgroundAndInstallsItOnOK(t *testing.T) {
	src := &fakeSource{update: domain.Update{Version: "1.1.0", URL: "u", AppSHA256: "e1"}}
	_, rec, inst, restarts, app := startWithUpdates(t, src, "en")
	settle(t, rec, app)
	m := app.doc.Modal()
	if m == nil || m.Title != "Update available" {
		t.Fatalf("no update dialog: %+v", m)
	}
	m.Confirm()
	for _, f := range rec.later { // the install runs on the next turn of the loop
		f()
	}
	if inst.got.Version != "1.1.0" || *restarts != 1 {
		t.Errorf("OK must install and restart: %+v restarts %d", inst.got, *restarts)
	}
}

func TestLaterPostponesAndAFailureIsShownWithoutRestarting(t *testing.T) {
	src := &fakeSource{update: domain.Update{Version: "1.1.0", AppSHA256: "e2"}}
	rt, rec, inst, restarts, app := startWithUpdates(t, src, "pt-BR")
	settle(t, rec, app)
	app.doc.Modal().Cancel()
	if rt.updates.State.Load().Dismissed != "e2" {
		t.Error("Depois must postpone this version")
	}
	inst.err = errors.New("disk full")
	rt.installUpdate(domain.Update{Version: "1.1.0"})
	if m := app.doc.Modal(); m == nil || *restarts != 0 {
		t.Errorf("the failure must be shown and nothing restarted: %v %d", m, *restarts)
	}
}

func TestTheOfferWaitsForADialogThatIsAlreadyOpen(t *testing.T) {
	src := &fakeSource{update: domain.Update{Version: "1.1.0", AppSHA256: "e3"}}
	_, rec, _, _, app := startWithUpdates(t, src, "en")
	mine := widget.NewModal("Mine")
	mine.Open(app.doc)
	for i := 0; i < 200 && len(rec.later) == 0; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	for i := 0; i < 5; i++ {
		later := rec.later
		rec.later = nil
		for _, f := range later {
			f()
		}
		time.Sleep(10 * time.Millisecond)
	}
	if app.doc.Modal() != mine {
		t.Error("the update dialog must not cover the person's own dialog")
	}
	mine.Close()
	settle(t, rec, app)
	if m := app.doc.Modal(); m == nil || m.Title != "Update available" {
		t.Error("once the dialog is closed the offer appears")
	}
}

func TestAReleaseWithoutTheAppChecksumOffersNothing(t *testing.T) {
	src := &fakeSource{update: domain.Update{Version: "9.0.0"}}
	u, _, _, _ := newUpdater("1.0.0", src)
	if _, ok := u.Check(); ok {
		t.Error("without the checksum nothing says the app changed")
	}
}

func TestCheckNeverOffersAnOlderReleaseToANewerBuild(t *testing.T) {
	src := &fakeSource{update: domain.Update{Version: "1.49.7", AppSHA256: "bbb"}}
	u, _, _, _ := newUpdater("1.49.8", src)
	if _, ok := u.Check(); ok {
		t.Error("a local build newer than the release must not be taken back to it")
	}
	same, _, _, _ := newUpdater("1.49.7", src)
	if _, ok := same.Check(); !ok {
		t.Error("the same version with another checksum is still an update")
	}
}
