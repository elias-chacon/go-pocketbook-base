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
	// AppSHA256 is the SHA-256 of the <id>.app of that build. It, not the version, says whether
	// the installed app is the published one: a release may change one app and not another.
	AppSHA256 string
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
	Dismissed string    `json:"dismissed,omitempty"` // the AppSHA256 of the build the person said "later" to
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
