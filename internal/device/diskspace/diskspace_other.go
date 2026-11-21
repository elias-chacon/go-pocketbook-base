//go:build !linux

package diskspace

func of(string) (Space, bool) { return Space{}, false }
