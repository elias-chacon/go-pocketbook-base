package domain

import "time"

// Easter is the date of Easter Sunday in a year (the Gregorian computus).
func Easter(year int) time.Time {
	a := year % 19
	b, c := year/100, year%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := (h+l-7*m+114)%31 + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// The books (canonical ids) the festive verses use.
const (
	bookIsaiah  = 23
	bookLament  = 25
	bookMatthew = 40
	bookLuke    = 42
	bookJohn    = 43
	bookActs    = 44
)

// FestiveVerse is the verse tied to a holy day, when the date is one: Christmas and
// its eve, New Year, Epiphany, Palm Sunday, Good Friday, Easter, the Ascension and
// Pentecost.
func FestiveVerse(day time.Time) (Ref, bool) {
	y, m, d := day.Date()
	switch {
	case m == time.December && d == 25:
		return Ref{bookLuke, 2, 11}, true
	case m == time.December && d == 24:
		return Ref{bookIsaiah, 9, 6}, true
	case m == time.January && d == 1:
		return Ref{bookLament, 3, 23}, true
	case m == time.January && d == 6:
		return Ref{bookMatthew, 2, 2}, true
	}
	date := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	switch int(date.Sub(Easter(y)).Hours() / 24) {
	case -7:
		return Ref{bookMatthew, 21, 9}, true // Palm Sunday
	case -2:
		return Ref{bookJohn, 19, 30}, true // Good Friday
	case 0:
		return Ref{bookMatthew, 28, 6}, true // Easter
	case 39:
		return Ref{bookActs, 1, 9}, true // the Ascension
	case 49:
		return Ref{bookActs, 2, 4}, true // Pentecost
	}
	return Ref{}, false
}

// DailySeed is a number that is the same all day and differs a lot from one day to
// the next (consecutive days must not pick neighbouring verses).
func DailySeed(day time.Time) int {
	y, m, d := day.Date()
	n := uint64(y*10000+int(m)*100+d) * 0x9E3779B97F4A7C15
	n ^= n >> 29
	return int(n >> 1 & 0x3fffffff)
}
