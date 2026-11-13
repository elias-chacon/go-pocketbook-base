//go:build linux && arm && cgo

// Package inkview is a minimal cgo binding to the PocketBook InkView library
// (libinkview), just enough for a vector-drawn app: event loop, framebuffer
// blit, screen refresh and exit.
package inkview

/*
#cgo CFLAGS: -pthread -I/SDK/usr/arm-obreey-linux-gnueabi/sysroot/usr/include/freetype2
#cgo LDFLAGS: -pthread -linkview
#include <stdlib.h>
#include <string.h>
#include "inkview.h"

extern int main_handler(int t, int p1, int p2);

// Exported by libinkview but missing from the public header.
void LeaveInkViewMain();
*/
import "C"

import (
	"fmt"
	"unsafe"
)

var app App

// Run starts the InkView event loop and blocks until the app exits.
func Run(a App) error {
	app = a
	C.InkViewMain(C.iv_handler(C.main_handler))
	return nil
}

// ScreenSize returns the current screen size (changes with orientation).
func ScreenSize() (w, h int) { return int(C.ScreenWidth()), int(C.ScreenHeight()) }

// Exit asks InkView to close the application.
//
// CloseApp asks InkView to deliver EVT_EXIT; LeaveInkViewMain makes the
// InkViewMain loop return so the process ends even if the event is not
// delivered (observed on PB741 firmware 6.x with CloseApp alone).
func Exit() {
	Logf("Exit: CloseApp + LeaveInkViewMain")
	C.CloseApp()
	C.LeaveInkViewMain()
}

// Repaint queues an EventShow so App.Draw is called again.
func Repaint() { C.Repaint() }

// DisablePanel turns the system status panel off. With the panel enabled
// (the default, panelType=1) InkView reserves PanelHeight() pixels and
// offsets the app canvas by that amount, so a full-screen drawing ends up
// shifted down on a PB741 (seen in a photo: ~137 px off-center). Without the
// panel the canvas is the whole screen. Call it in Init, before drawing.
func DisablePanel() { C.SetPanelType(0) }

// Info describes the screen/canvas/panel state, for diagnostics.
func Info() string {
	s := fmt.Sprintf("screen=%dx%d orientation=%d panelHeight=%d panelType=%d model=%q",
		C.ScreenWidth(), C.ScreenHeight(), C.GetOrientation(), C.PanelHeight(), C.GetPanelType(),
		C.GoString(C.GetDeviceModel()))
	if cv := C.GetCanvas(); cv != nil {
		s += fmt.Sprintf(" canvas=%dx%d scanline=%d depth=%d clip=(%d,%d)-(%d,%d)",
			cv.width, cv.height, cv.scanline, cv.depth, cv.clipx1, cv.clipy1, cv.clipx2, cv.clipy2)
	}
	return s
}

// Blit draws an 8-bit grayscale buffer (pix, stride bytes per row) at 0,0 and
// refreshes the e-ink panel.
func Blit(pix []byte, w, h, stride int) {
	n := h * stride
	hdr := C.size_t(unsafe.Sizeof(C.ibitmap{}))
	mem := C.malloc(hdr + C.size_t(n))
	defer C.free(mem)

	bm := (*C.ibitmap)(mem)
	bm.width = C.ushort(w)
	bm.height = C.ushort(h)
	bm.depth = 8
	bm.scanline = C.ushort(stride)
	C.memcpy(unsafe.Pointer(uintptr(mem)+uintptr(hdr)), unsafe.Pointer(&pix[0]), C.size_t(n))

	C.ClearScreen()
	C.DrawBitmap(0, 0, bm)
	C.FullUpdate()
}

//export goMainHandler
func goMainHandler(t, p1, p2 C.int) C.int {
	logEvent(int(t), int(p1), int(p2))
	switch int(t) {
	case C.EVT_INIT:
		app.Init()
		return 1
	case C.EVT_SHOW:
		app.Draw()
		return 1
	case C.EVT_KEYPRESS:
		if app.Key(int(p1)) {
			return 1
		}
	case C.EVT_POINTERUP, C.EVT_TOUCHUP:
		// Some firmwares report taps as touch events, others as pointer events.
		if app.Pointer(int(p1), int(p2)) {
			return 1
		}
	case C.EVT_EXIT:
		return 1
	}
	return 0
}

// Key codes (subset of inkview.h).
const (
	KeyBack = 0x1b
	KeyOK   = 0x0a
	KeyMenu = 0x17
	KeyPrev = 0x18
	KeyNext = 0x19
)
