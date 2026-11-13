package widget

import "strings"

// Masks describe a digit pattern: '0' and '#' stand for one digit, every other
// character is a literal that is inserted automatically. Example: the mask
// "(00) 00000-0000" turns the digits "11987654321" into "(11) 98765-4321".

func isSlot(r rune) bool { return r == '0' || r == '#' }

// MaskSlots is the number of digits the mask accepts.
func MaskSlots(mask string) int {
	n := 0
	for _, r := range mask {
		if isSlot(r) {
			n++
		}
	}
	return n
}

// OnlyDigits drops everything that is not 0-9.
func OnlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ApplyMask formats digits with mask. It stops where the digits run out, so a
// partial value shows no dangling literals ("(11) 98" for "1198").
func ApplyMask(mask, digits string) string {
	digits = OnlyDigits(digits)
	var b strings.Builder
	i := 0
	for _, m := range mask {
		if i >= len(digits) {
			break
		}
		if isSlot(m) {
			b.WriteByte(digits[i])
			i++
		} else {
			b.WriteRune(m)
		}
	}
	return b.String()
}

// TrimToMask keeps at most as many digits as the mask has slots.
func TrimToMask(mask, digits string) string {
	digits = OnlyDigits(digits)
	if n := MaskSlots(mask); len(digits) > n {
		return digits[:n]
	}
	return digits
}
