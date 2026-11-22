package inkview

// UpdateMode is how a part of the panel is refreshed after it was drawn. They are the update
// functions of InkView, from the default one to the fast ones that trade quality for speed (the
// engineering app of this repository measures them on the reader).
type UpdateMode int

// The update modes, in the order the engineering app lists them.
const (
	UpdatePartial UpdateMode = iota // PartialUpdate: the default
	UpdateDynamic                   // DynamicUpdate
	UpdateA2                        // DynamicUpdateA2: the fastest, two levels (dithered), needs ExitA2 after
	UpdateDU4                       // PartialUpdateDU4: four levels
	UpdateBW                        // PartialUpdateBW: black and white
	UpdateHQ                        // PartialUpdateHQ: the best quality, slow
	UpdateFull                      // FullUpdate: the whole panel, flashing
	updateModes
)

// UpdateModes is how many update modes there are.
const UpdateModes = int(updateModes)

var updateNames = [...]string{"partial", "dynamic", "a2", "du4", "bw", "hq", "full"}

// String is the short name of the mode (the one the engineering log uses).
func (m UpdateMode) String() string {
	if m < 0 || int(m) >= len(updateNames) {
		return "unknown"
	}
	return updateNames[m]
}

// inside reports whether the rectangle (x, y, w, h) lies inside a buffer of n bytes with the given
// stride and bytes per pixel: a drawing call with a rectangle outside its picture draws nothing
// instead of reading past the end.
func inside(n, stride, bpp, x, y, w, h int) bool {
	if x < 0 || y < 0 || w <= 0 || h <= 0 || stride <= 0 || (x+w)*bpp > stride {
		return false
	}
	return (y+h-1)*stride+(x+w)*bpp <= n
}
