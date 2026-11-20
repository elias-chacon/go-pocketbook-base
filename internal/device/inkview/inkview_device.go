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
extern void c_keyboard_handler(char *text);

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

// BlitRect draws only the rectangle (x, y, w, h) of an 8-bit grayscale buffer
// (stride bytes per row, the whole screen) and refreshes just that part of the
// e-ink panel: much faster than Blit for a small change.
func BlitRect(pix []byte, stride, x, y, w, h int) {
	hdr := C.size_t(unsafe.Sizeof(C.ibitmap{}))
	mem := C.malloc(hdr + C.size_t(w*h))
	defer C.free(mem)

	bm := (*C.ibitmap)(mem)
	bm.width = C.ushort(w)
	bm.height = C.ushort(h)
	bm.depth = 8
	bm.scanline = C.ushort(w)
	for row := 0; row < h; row++ {
		C.memcpy(unsafe.Pointer(uintptr(mem)+uintptr(hdr)+uintptr(row*w)), unsafe.Pointer(&pix[(y+row)*stride+x]), C.size_t(w))
	}

	C.DrawBitmap(C.int(x), C.int(y), bm)
	C.PartialUpdate(C.int(x), C.int(y), C.int(w), C.int(h))
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
		return boolToInt(app.Key(int(p1)))
	case C.EVT_POINTERDOWN:
		return boolToInt(app.Pointer(PointerDown, int(p1), int(p2)))
	case C.EVT_POINTERMOVE:
		return boolToInt(app.Pointer(PointerMove, int(p1), int(p2)))
	case C.EVT_POINTERUP:
		return boolToInt(app.Pointer(PointerUp, int(p1), int(p2)))
	case C.EVT_EXIT:
		return 1
	}
	return 0
}

func boolToInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// ---- system keyboard ----

var (
	kbDone  func(text string, ok bool)
	kbBuf   *C.char
	kbTitle *C.char
)

// OpenKeyboard opens InkView's on-screen keyboard. done receives the typed
// text, or ok=false if the user cancelled. flags are KeyboardNormal/Numeric.
func OpenKeyboard(title, initial string, maxLen, flags int, done func(text string, ok bool)) {
	if maxLen <= 0 {
		maxLen = 1024
	}
	releaseKeyboard()
	kbDone = done
	kbBuf = (*C.char)(C.calloc(C.size_t(maxLen+1), 1))
	cInitial := C.CString(initial)
	C.strncpy(kbBuf, cInitial, C.size_t(maxLen))
	C.free(unsafe.Pointer(cInitial))
	kbTitle = C.CString(title)
	C.OpenKeyboard(kbTitle, kbBuf, C.int(maxLen), C.int(flags), C.iv_keyboardhandler(C.c_keyboard_handler))
}

func releaseKeyboard() {
	if kbBuf != nil {
		C.free(unsafe.Pointer(kbBuf))
		C.free(unsafe.Pointer(kbTitle))
		kbBuf, kbTitle = nil, nil
	}
}

//export goKeyboardHandler
func goKeyboardHandler(text *C.char) {
	done := kbDone
	kbDone = nil
	s, ok := "", text != nil
	if ok {
		s = C.GoString(text)
	}
	releaseKeyboard()
	if done != nil {
		done(s, ok)
	}
}

// GlobalConfigString reads a key of the reader's global settings (global.cfg, the file where the
// firmware keeps "language=pt", "ttslanguage=..." and so on) through InkView, "" when it is not set.
func GlobalConfigString(name string) string {
	cfg := C.GetGlobalConfig()
	if cfg == nil {
		return ""
	}
	key := C.CString(name)
	defer C.free(unsafe.Pointer(key))
	empty := C.CString("")
	defer C.free(unsafe.Pointer(empty))
	return C.GoString(C.ReadString(cfg, key, empty))
}

// NetConnect brings the default network (Wi-Fi) up and blocks until it is
// connected or it fails; it returns nil when the reader already was. The reader
// does not keep Wi-Fi connected for an app: without this call a request fails
// with "network is unreachable".
func NetConnect() error {
	rc := int(C.NetConnect(nil))
	Logf("NetConnect -> %d", rc)
	if rc != int(C.NET_OK) {
		return fmt.Errorf("inkview: NetConnect failed (code %d): is Wi-Fi on and a network known?", rc)
	}
	return nil
}
