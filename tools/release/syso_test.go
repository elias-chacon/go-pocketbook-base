package main

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"image"
	"image/png"
	"testing"
)

func square(size int) image.Image { return image.NewGray(image.Rect(0, 0, size, size)) }

// resource walks the section like the loader does: type -> name -> language -> data.
func resource(t *testing.T, section []byte, typ, name uint32) []byte {
	t.Helper()
	le := binary.LittleEndian
	find := func(dirAt uint32, id uint32) uint32 {
		n := int(le.Uint16(section[dirAt+14:]))
		for i := 0; i < n; i++ {
			e := dirAt + 16 + uint32(i)*8
			if le.Uint32(section[e:]) == id {
				return le.Uint32(section[e+4:]) &^ 0x80000000
			}
		}
		t.Fatalf("id %d not found in the directory at %d", id, dirAt)
		return 0
	}
	lang := find(find(0, typ), name)
	entry := find(lang, 0)
	rva, size := le.Uint32(section[entry:]), le.Uint32(section[entry+4:])
	return section[rva : rva+size]
}

func TestSysoIsAValidObjectWithTheIconResources(t *testing.T) {
	data, err := sysoFor(square)
	if err != nil {
		t.Fatal(err)
	}
	f, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a COFF object: %v", err)
	}
	if f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || len(f.Sections) != 1 || f.Sections[0].Name != ".rsrc" {
		t.Fatalf("%x %v", f.Machine, f.Sections)
	}
	sec := f.Sections[0]
	if int(sec.NumberOfRelocations) != len(icoSizes)+1 {
		t.Errorf("one relocation per data entry: %d", sec.NumberOfRelocations)
	}
	if len(f.Symbols) != 1 || f.Symbols[0].Name != ".rsrc" {
		t.Errorf("the section symbol: %v", f.Symbols)
	}
	section, err := sec.Data()
	if err != nil {
		t.Fatal(err)
	}

	checkIcons(t, section)
	checkGroup(t, section)
}

// checkIcons: every size is a PNG of its own edge.
func checkIcons(t *testing.T, section []byte) {
	t.Helper()
	for i, edge := range icoSizes {
		img, err := png.Decode(bytes.NewReader(resource(t, section, rtIcon, uint32(i+1))))
		if err != nil || img.Bounds().Dx() != edge {
			t.Errorf("icon %d: %v %v", i+1, err, img)
		}
	}
}

// checkGroup: the group lists the icons in the same order, pointing at their ids.
func checkGroup(t *testing.T, section []byte) {
	t.Helper()
	group := resource(t, section, rtGroupIcon, 1)
	le := binary.LittleEndian
	if le.Uint16(group[2:]) != 1 || int(le.Uint16(group[4:])) != len(icoSizes) {
		t.Fatalf("the group header: %v", group[:6])
	}
	for i := range icoSizes {
		e := group[6+i*14:]
		size := uint32(len(resource(t, section, rtIcon, uint32(i+1))))
		if int(le.Uint16(e[12:])) != i+1 || size != le.Uint32(e[8:]) {
			t.Errorf("entry %d: %v", i, e[:14])
		}
	}
	if group[6] != 0 || group[7] != 0 {
		t.Error("256 is stored as 0")
	}
}

func TestEveryRelocationPointsAtADataEntry(t *testing.T) {
	data := buildSyso([][]byte{{1, 2, 3}}, []int{16})
	f, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	sec := f.Sections[0]
	le := binary.LittleEndian
	raw := data[sec.Offset : sec.Offset+sec.Size]
	relocs := data[sec.PointerToRelocations:]
	for i := 0; i < int(sec.NumberOfRelocations); i++ {
		r := relocs[i*10:]
		at := le.Uint32(r)
		if le.Uint16(r[8:]) != relAMD64Addr32NB || le.Uint32(r[4:]) != 0 {
			t.Errorf("reloc %d: type/symbol", i)
		}
		rva := le.Uint32(raw[at:])
		if int(rva) >= len(raw) || int(rva)%4 != 0 {
			t.Errorf("reloc %d: the data is at %d of %d", i, rva, len(raw))
		}
	}
}

func TestSysoFailsWhenAnIconCannotBeEncoded(t *testing.T) {
	if _, err := sysoFor(func(int) image.Image { return image.NewGray(image.Rect(0, 0, 0, 0)) }); err == nil {
		t.Error("an empty image cannot be a PNG")
	}
}
