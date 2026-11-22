//go:build linux && arm && cgo

package inkview

/*
#cgo CFLAGS: -pthread -I/SDK/usr/arm-obreey-linux-gnueabi/sysroot/usr/include/freetype2
#cgo LDFLAGS: -pthread -linkview
#include <stdlib.h>
#include <string.h>
#include "inkview.h"
*/
import "C"

import "unsafe"

// Update refreshes the rectangle (x, y, w, h) of the panel with the given mode. A mode other than A2
// first leaves the A2 state if the panel is in it, as the SDK's own scroll view does.
func Update(mode UpdateMode, x, y, w, h int) {
	if mode != UpdateA2 && C.IsInA2Update() != 0 {
		C.ExitUpdateA2()
	}
	cx, cy, cw, ch := C.int(x), C.int(y), C.int(w), C.int(h)
	switch mode {
	case UpdateDynamic:
		C.DynamicUpdate(cx, cy, cw, ch)
	case UpdateA2:
		C.DynamicUpdateA2(cx, cy, cw, ch)
	case UpdateDU4:
		C.PartialUpdateDU4(cx, cy, cw, ch)
	case UpdateBW:
		C.PartialUpdateBW(cx, cy, cw, ch)
	case UpdateHQ:
		C.PartialUpdateHQ(cx, cy, cw, ch)
	case UpdateFull:
		C.FullUpdate()
	default:
		C.PartialUpdate(cx, cy, cw, ch)
	}
}

// bitmap makes an InkView bitmap of w x h pixels of the given depth (bytes per pixel = depth/8, or
// the packed row when depth is 8) and returns it with the memory to free.
func bitmap(w, h, depth, scanline int) (*C.ibitmap, unsafe.Pointer) {
	hdr := C.size_t(unsafe.Sizeof(C.ibitmap{}))
	mem := C.malloc(hdr + C.size_t(scanline*h))
	bm := (*C.ibitmap)(mem)
	bm.width = C.ushort(w)
	bm.height = C.ushort(h)
	bm.depth = C.ushort(depth)
	bm.scanline = C.ushort(scanline)
	return bm, mem
}

// BlitGrayMode draws the part (x, y, w, h) of an 8-bit grayscale buffer and refreshes it with mode.
func BlitGrayMode(pix []byte, stride, x, y, w, h int, mode UpdateMode) {
	if !inside(len(pix), stride, 1, x, y, w, h) {
		return
	}
	bm, mem := bitmap(w, h, 8, w)
	defer C.free(mem)
	data := unsafe.Pointer(uintptr(mem) + unsafe.Sizeof(C.ibitmap{}))
	for row := 0; row < h; row++ {
		C.memcpy(unsafe.Pointer(uintptr(data)+uintptr(row*w)), unsafe.Pointer(&pix[(y+row)*stride+x]), C.size_t(w))
	}
	C.DrawBitmap(C.int(x), C.int(y), bm)
	Update(mode, x, y, w, h)
}

// ColorBGR is the byte order of a 24-bit bitmap: true when the blue byte comes first (what the
// framebuffer of a color PocketBook holds; checked against the firmware of the PB741 in the pbemu
// emulator: the color bars of the engineering app came out with the right colors).
var ColorBGR = true

// BlitColorMode draws the part (x, y, w, h) of an RGBA buffer (4 bytes per pixel) as a 24-bit
// bitmap and refreshes it with mode.
func BlitColorMode(rgba []byte, stride, x, y, w, h int, mode UpdateMode) {
	if !inside(len(rgba), stride, 4, x, y, w, h) {
		return
	}
	bm, mem := bitmap(w, h, 24, w*3)
	defer C.free(mem)
	data := (*[1 << 30]byte)(unsafe.Pointer(uintptr(mem) + unsafe.Sizeof(C.ibitmap{})))[: w*3*h : w*3*h]
	for row := 0; row < h; row++ {
		src := rgba[(y+row)*stride+x*4:]
		dst := data[row*w*3:]
		for col := 0; col < w; col++ {
			r, g, b := src[col*4], src[col*4+1], src[col*4+2]
			if ColorBGR {
				r, b = b, r
			}
			dst[col*3], dst[col*3+1], dst[col*3+2] = r, g, b
		}
	}
	C.DrawBitmap(C.int(x), C.int(y), bm)
	Update(mode, x, y, w, h)
}

// ExitA2 leaves the A2 state with a final, clean update of what was drawn in it.
func ExitA2() {
	if C.IsInA2Update() != 0 {
		C.ExitUpdateA2()
	}
}

// WaitUpdate blocks until the panel finished the updates asked for.
func WaitUpdate() { C.WaitForUpdateComplete() }

// HardwareDepth is the bits per pixel of the panel's framebuffer (8: gray, 24: color).
func HardwareDepth() int { return int(C.GetHardwareDepth()) }

// Waveform is the name of the waveform file of the panel ("" when unknown).
func Waveform() string {
	name := C.GetWaveformFilename()
	if name == nil {
		return ""
	}
	return C.GoString(name)
}
