// Package version implements the project's MAJOR.MINOR.BUILD versioning.
//
//   - BUILD is incremented on every build.
//   - MINOR is incremented when a task is finished (BUILD restarts at 0).
//   - MAJOR is incremented only for a big change (MINOR and BUILD restart at 0).
//
// The repository's .version names the release (the tag); every app has its own apps/<id>/.version,
// bumped only when that app changed (tools/bumpversion), so a release does not make the apps that
// did not change look updated. The first line of a .version file is the version; what follows is the
// tool's own note.
//
// The reference is the .version file at the repository root. The binary gets it
// through -ldflags "-X go-pocketbook-base/internal/kernel/version.Value=<version>".
package version

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Value is the version compiled into the binary (set with -ldflags -X).
var Value = "dev"

// Kind selects which part of the version to increment.
type Kind string

// The kinds of bump.
const (
	Build Kind = "build"
	Minor Kind = "minor"
	Major Kind = "major"
)

// Version is a MAJOR.MINOR.BUILD version.
type Version struct{ Major, Minor, Build int }

// Parse reads "MAJOR.MINOR.BUILD".
func Parse(s string) (Version, error) {
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("version %q: want MAJOR.MINOR.BUILD", s)
	}
	var n [3]int
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 {
			return Version{}, fmt.Errorf("version %q: %q is not a non-negative number", s, p)
		}
		n[i] = v
	}
	return Version{n[0], n[1], n[2]}, nil
}

// Less reports whether v comes before o.
func (v Version) Less(o Version) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	return v.Build < o.Build
}

// Older reports whether version a is older than version b. Text that is not a version (a "dev"
// build) is never older: nothing can be said, so the caller decides some other way.
func Older(a, b string) bool {
	x, err1 := Parse(a)
	y, err2 := Parse(b)
	return err1 == nil && err2 == nil && x.Less(y)
}

// String formats the version as MAJOR.MINOR.BUILD.
func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Build) }

// Bump returns the next version for the given kind.
func (v Version) Bump(k Kind) (Version, error) {
	switch k {
	case Build:
		v.Build++
	case Minor:
		v.Minor, v.Build = v.Minor+1, 0
	case Major:
		v.Major, v.Minor, v.Build = v.Major+1, 0, 0
	default:
		return v, fmt.Errorf("unknown bump kind %q (use build, minor or major)", k)
	}
	return v, nil
}

// ReadFile reads the version on the first line of path.
func ReadFile(path string) (Version, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Version{}, err
	}
	first := strings.SplitN(strings.ReplaceAll(string(data), "\r", ""), "\n", 2)[0]
	return Parse(first)
}

// BumpFile increments the version stored in path and returns the new one. The lines after the
// first are kept.
func BumpFile(path string, k Kind) (Version, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Version{}, err
	}
	cur, err := ReadFile(path)
	if err != nil {
		return Version{}, err
	}
	next, err := cur.Bump(k)
	if err != nil {
		return Version{}, err
	}
	rest := ""
	if i := strings.Index(string(data), "\n"); i >= 0 {
		rest = string(data)[i+1:]
	}
	return next, os.WriteFile(path, []byte(next.String()+"\n"+rest), 0o644)
}
