package application

import (
	"time"

	"go-pocketbook-base/internal/platform/domain"
)

// DefaultCheckEvery is how long a successful check keeps the app from asking the network
// again: opening apps one after the other (the Launcher, then an app, then back) would
// otherwise ask GitHub at every step.
const DefaultCheckEvery = 10 * time.Minute

// Updater is the use case "keep this app up to date": find out, in the background, whether a
// newer build is published, and replace the app's file with it when the person agrees. The
// same for every app of the ecosystem and for the Launcher (platform.Run wires it).
type Updater struct {
	ID      string // the app
	Current string // the version running
	Source  domain.UpdateSource
	Install domain.UpdateInstaller
	State   domain.UpdateStateStore
	// Restart starts the new build in place of the running one; it returns only on failure.
	Restart func() error

	CheckEvery time.Duration    // DefaultCheckEvery when zero
	Now        func() time.Time // time.Now when nil (tests)
}

func (u Updater) now() time.Time {
	if u.Now != nil {
		return u.Now()
	}
	return time.Now()
}

func (u Updater) every() time.Duration {
	if u.CheckEvery > 0 {
		return u.CheckEvery
	}
	return DefaultCheckEvery
}

// Check asks whether there is a newer build the person has not postponed. It blocks on the
// network, so the runtime calls it away from the event loop. Offline (or any failure to find
// out) is not an error to show: it answers no, and the next opening tries again.
func (u Updater) Check() (domain.Update, bool) {
	if !domain.IsRelease(u.Current) {
		return domain.Update{}, false // a developer's build is never replaced
	}
	st := u.State.Load()
	latest := st.Available
	if st.CheckedAt.IsZero() || u.now().Sub(st.CheckedAt) >= u.every() {
		found, err := u.Source.Latest(u.ID)
		if err != nil {
			return domain.Update{}, false // offline, or no release: try again next time
		}
		latest = found
		st.CheckedAt, st.Available = u.now(), found
		_ = u.State.Save(st) // the memory is only a courtesy to the network
	}
	if !domain.IsNewer(u.Current, latest.Version) || latest.Version == st.Dismissed {
		return domain.Update{}, false
	}
	return latest, true
}

// Postpone remembers that the person said "later" to this version, so it is not offered again
// (a still newer one will be).
func (u Updater) Postpone(up domain.Update) {
	st := u.State.Load()
	st.Dismissed = up.Version
	_ = u.State.Save(st)
}

// Apply downloads the update, replaces the app's file and restarts it. It returns only when
// something failed; the old build stays in place then.
func (u Updater) Apply(up domain.Update) error {
	if err := u.Install.Install(u.ID, up); err != nil {
		return err
	}
	return u.Restart()
}
