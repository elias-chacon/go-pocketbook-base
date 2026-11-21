// Package diskspace tells how much room a storage has: the free and the total bytes of the volume
// a path is on. On a system where it cannot be known (anything but Linux) it says so.
package diskspace

// Space is the size of a volume in bytes.
type Space struct{ Total, Free uint64 }

// Of returns the space of the volume that holds path; ok is false when it cannot be known.
func Of(path string) (s Space, ok bool) { return of(path) }
