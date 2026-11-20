package domain

import "time"

// ChapterRef names one chapter of a book.
type ChapterRef struct{ Book, Chapter int }

// Catalog is where the Bible comes from (the Midvash API): the list of books and
// versions, and the text of chapters.
type Catalog interface {
	Books() ([]Book, error)
	Versions() ([]Version, error)
	// Chapters returns the text of the given chapters of a version, in few
	// requests. A chapter the version does not have is left out.
	Chapters(version string, books []Book, refs []ChapterRef) ([]ChapterText, error)
}

// Library is the Bible on the reader: what was downloaded, what was read and how.
// Every query reads what it needs, a chapter at a time, never a whole version.
type Library interface {
	// Catalog data
	SaveBooks(books []Book) error
	Books() ([]Book, error)
	SaveVersions(versions []Version) error
	Versions() ([]Version, error)
	Version(slug string) (Version, bool, error)

	// Text
	SaveChapters(chapters []ChapterText) error
	// StoredChapters is the chapters of a version already on the reader.
	StoredChapters(version string) (map[ChapterRef]bool, error)
	MarkDownloaded(version string) error
	DeleteVersion(version string) error
	Chapter(version string, book, chapter int) (ChapterText, error)
	// Previews is the start of the first verse of each chapter of a book (chapter -> text).
	Previews(version string, book, max int) (map[int]string, error)
	VerseText(version string, ref Ref) (string, bool, error)
	// Search finds the verses that have every word (already folded), at most limit.
	Search(version string, words []string, limit int) ([]Hit, error)
	// PickVerse is the n-th verse of a version, counting from zero (n is reduced to the verses there are).
	PickVerse(version string, n int) (Ref, error)

	// Progress
	MarkRead(refs []Ref, at time.Time) error
	IsRead(ref Ref) (bool, error)
	// ReadCount is how many verses of the version have been read, and how many it has.
	ReadCount(version string) (read, total int, err error)

	// State
	SavePosition(p Position) error
	Position() (Position, error)
	LoadPreferences() (Preferences, bool, error)
	SavePreferences(p Preferences) error
	// Daily is the verse of a day and whether it was read.
	Daily(day string) (DailyVerse, bool, error)
	SaveDaily(d DailyVerse) error
}

// DailyVerse is the verse chosen for a day.
type DailyVerse struct {
	Day  string // 2006-01-02
	Ref  Ref
	Read bool
}

// Speaker reads a text aloud.
type Speaker interface {
	Available() bool
	Speak(text string) error
	Stop()
}
