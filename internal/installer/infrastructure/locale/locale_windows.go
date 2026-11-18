//go:build windows

package locale

import (
	"syscall"
	"unsafe"
)

// fromSystem asks Windows for the user's locale name (e.g. "pt-BR").
func fromSystem() string {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")
	buf := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH
	n, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
