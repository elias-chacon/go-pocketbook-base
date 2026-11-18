package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
)

// A .syso is a COFF object that the Go linker puts into a Windows executable.
// This one holds only a resource section (.rsrc) with the program's icon, so
// Explorer shows it: one RT_ICON per size (PNG data, as Windows Vista and later
// read it) and one RT_GROUP_ICON that lists them.
//
// The resource tree is Type -> Name -> Language -> data, and every data entry
// points at its bytes by an RVA, which the linker relocates (ADDR32NB against
// the section symbol), hence one relocation per data entry.
const (
	coffMachineAMD64 = 0x8664
	rtIcon           = 3
	rtGroupIcon      = 14
	langNeutral      = 0
	relAMD64Addr32NB = 3
	scnData          = 0x40000040 // initialized data, readable
	dirHeader        = 16
	dirEntry         = 8
	dataEntry        = 16
)

// icoSizes are the edges of the icons embedded in the executable.
var icoSizes = []int{256, 64, 48, 32, 16}

// sysoFor builds the .syso of an executable whose icon is drawn by render.
func sysoFor(render func(size int) image.Image) ([]byte, error) {
	var pngs [][]byte
	var edges []int
	for _, size := range icoSizes {
		var b bytes.Buffer
		if err := png.Encode(&b, render(size)); err != nil {
			return nil, err
		}
		pngs = append(pngs, b.Bytes())
		edges = append(edges, size)
	}
	return buildSyso(pngs, edges), nil
}

// buildSyso lays out the resource section for the PNG icons (edges[i] is the
// side of pngs[i]) and wraps it in a COFF object.
func buildSyso(pngs [][]byte, edges []int) []byte {
	n := len(pngs)
	le := binary.LittleEndian

	// The group directory (GRPICONDIR + one GRPICONDIRENTRY per icon).
	group := &bytes.Buffer{}
	binary.Write(group, le, []uint16{0, 1, uint16(n)})
	for i, p := range pngs {
		side := byte(edges[i] % 256) // 256 is stored as 0
		group.Write([]byte{side, side, 0, 0})
		binary.Write(group, le, []uint16{1, 32})
		binary.Write(group, le, uint32(len(p)))
		binary.Write(group, le, uint16(i+1))
	}
	blobs := append(append([][]byte{}, pngs...), group.Bytes()) // data entry k <-> blob k

	// Offsets inside the section.
	iconType := dirHeader + 2*dirEntry
	groupType := iconType + dirHeader + n*dirEntry
	langDirs := groupType + dirHeader + dirEntry
	dataEntries := langDirs + (n+1)*(dirHeader+dirEntry)
	blobStart := dataEntries + (n+1)*dataEntry

	blobOff := make([]int, n+1)
	off := blobStart
	for k, b := range blobs {
		blobOff[k] = off
		off += (len(b) + 3) &^ 3
	}
	section := make([]byte, off)

	put32 := func(at int, v uint32) { le.PutUint32(section[at:], v) }
	put16 := func(at int, v uint16) { le.PutUint16(section[at:], v) }
	const subdir = 0x80000000
	dir := func(at, ids int) { put16(at+14, uint16(ids)) } // NumberOfIdEntries

	dir(0, 2) // root: the types, by ascending id
	put32(dirHeader, rtIcon)
	put32(dirHeader+4, subdir|uint32(iconType))
	put32(dirHeader+8, rtGroupIcon)
	put32(dirHeader+12, subdir|uint32(groupType))

	dir(iconType, n) // RT_ICON: one name (id) per icon
	for k := 0; k < n; k++ {
		e := iconType + dirHeader + k*dirEntry
		put32(e, uint32(k+1))
		put32(e+4, subdir|uint32(langDirs+k*(dirHeader+dirEntry)))
	}
	dir(groupType, 1) // RT_GROUP_ICON: one name
	put32(groupType+dirHeader, 1)
	put32(groupType+dirHeader+4, subdir|uint32(langDirs+n*(dirHeader+dirEntry)))

	relocAt := make([]int, 0, n+1)
	for k := 0; k <= n; k++ { // one language directory and one data entry each
		l := langDirs + k*(dirHeader+dirEntry)
		d := dataEntries + k*dataEntry
		dir(l, 1)
		put32(l+dirHeader, langNeutral)
		put32(l+dirHeader+4, uint32(d))
		put32(d, uint32(blobOff[k])) // RVA, relocated by the linker
		put32(d+4, uint32(len(blobs[k])))
		copy(section[blobOff[k]:], blobs[k])
		relocAt = append(relocAt, d)
	}

	// COFF: file header, one section header, the section data, its relocations,
	// the symbol table (one symbol: the section) and an empty string table.
	const headers = 20 + 40
	relocOff := headers + len(section)
	symOff := relocOff + 10*len(relocAt)
	out := &bytes.Buffer{}
	binary.Write(out, le, struct {
		Machine, Sections          uint16
		Time, SymbolTable, Symbols uint32
		OptionalHeader, Flags      uint16
	}{coffMachineAMD64, 1, 0, uint32(symOff), 1, 0, 0})
	name := [8]byte{'.', 'r', 's', 'r', 'c'}
	out.Write(name[:])
	binary.Write(out, le, struct {
		VirtualSize, VirtualAddress, RawSize, RawPointer, RelocPointer, LinePointer uint32
		Relocs, Lines                                                               uint16
		Flags                                                                       uint32
	}{0, 0, uint32(len(section)), headers, uint32(relocOff), 0, uint16(len(relocAt)), 0, scnData})
	out.Write(section)
	for _, at := range relocAt {
		binary.Write(out, le, struct {
			Address, Symbol uint32
			Type            uint16
		}{uint32(at), 0, relAMD64Addr32NB})
	}
	sym := [8]byte{'.', 'r', 's', 'r', 'c'}
	out.Write(sym[:])
	binary.Write(out, le, struct {
		Value   uint32
		Section int16
		Type    uint16
		Class   uint8
		Aux     uint8
	}{0, 1, 0, 3, 0}) // static symbol at the start of section 1
	binary.Write(out, le, uint32(4)) // string table: only its own size
	return out.Bytes()
}
