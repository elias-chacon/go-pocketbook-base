//go:build !(linux && arm && cgo)

package inkview

import "errors"

// ErrNotOnDevice is returned by Run when built for a non-PocketBook target.
var ErrNotOnDevice = errors.New("inkview: only available when built for PocketBook (linux/arm + cgo); use cmd/preview on desktop")

const (
	KeyBack = 0x1b
	KeyOK   = 0x0a
	KeyMenu = 0x17
	KeyPrev = 0x18
	KeyNext = 0x19
)

func Run(App) error                     { return ErrNotOnDevice }
func ScreenSize() (w, h int)            { return 0, 0 }
func Exit()                             {}
func Repaint()                          {}
func Info() string                      { return "" }
func DisablePanel()                     {}
func Blit(pix []byte, w, h, stride int) {}
