package domain

import (
	"strings"
	"time"

	"go-pocketbook-base/internal/kernel/version"
)

// Update is a newer build of an app, as the published release announces it.
type Update struct {
	Version string // MAJOR.MINOR.BUILD
	URL     string // of the zip with <id>.app and app.json
	Size    int64  // of the zip (0 when unknown)
	SHA256  string // of the zip, hex ("" when unknown)
}

// UpdateSource tells which build of an app is the latest one published.
type UpdateSource interface {
	// Latest returns the newest build of the app, whether it is newer or not; the
	// error is any failure to find out (no network, no release).
	Latest(appID string) (Update, error)
}

// UpdateInstaller replaces the app's own file with an update.
type UpdateInstaller interface {
	Install(appID string, u Update) error
}

// UpdateState is what the app remembers between runs about updates, so that it does not
// ask the network at every opening or nag about a version the person postponed.
type UpdateState struct {
	CheckedAt time.Time `json:"checkedAt"`
	Available Update    `json:"available"`           // the newest build the last check found (Version "" when none)
	Dismissed string    `json:"dismissed,omitempty"` // the version the person said "later" to
}

// UpdateStateStore keeps the UpdateState.
type UpdateStateStore interface {
	Load() UpdateState
	Save(UpdateState) error
}

// IsRelease reports whether v is a released version (MAJOR.MINOR.BUILD) and not a
// developer's build ("dev").
func IsRelease(v string) bool {
	_, err := version.Parse(strings.TrimPrefix(v, "v"))
	return err == nil
}

// IsNewer reports whether candidate is a later version than current. A build that is not
// a release ("dev") or a text that is not a version never counts as newer or older:
// nothing is offered, so a developer's build is never replaced.
func IsNewer(current, candidate string) bool {
	cur, err := version.Parse(strings.TrimPrefix(current, "v"))
	if err != nil {
		return false
	}
	next, err := version.Parse(strings.TrimPrefix(candidate, "v"))
	if err != nil {
		return false
	}
	switch {
	case next.Major != cur.Major:
		return next.Major > cur.Major
	case next.Minor != cur.Minor:
		return next.Minor > cur.Minor
	}
	return next.Build > cur.Build
}
