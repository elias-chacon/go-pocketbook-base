//go:build linux

package diskspace

import "syscall"

func of(path string) (Space, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Space{}, false
	}
	size := uint64(st.Bsize)
	return Space{Total: uint64(st.Blocks) * size, Free: uint64(st.Bavail) * size}, true
}
