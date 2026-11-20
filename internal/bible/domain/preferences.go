package domain

// Reading sizes, in design pixels (the toolkit's body text is 32).
const (
	MinFontSize     = 24
	MaxFontSize     = 56
	FontSizeStep    = 2
	DefaultFontSize = 34
)

// Preferences are the choices of the person, kept between runs.
type Preferences struct {
	// Lang is the language of the interface ("pt-BR", "en"); "" follows the reader.
	Lang string `json:"lang"`
	// VersionLang and Version are the language and the translation in use.
	VersionLang string `json:"versionLang"`
	Version     string `json:"version"`
	FontSize    int    `json:"fontSize"`
	// Font is the typeface ("go", "medium", "mono"; see gfx.Family).
	Font string `json:"font"`
	// Invert shows light text on a dark page.
	Invert bool `json:"invert"`
	// Speech enables reading aloud.
	Speech bool `json:"speech"`
	// ChapterTitles shows a preview of each chapter in the list of chapters.
	ChapterTitles bool `json:"chapterTitles"`
	// Alphabetical lists the books by name instead of the canonical order.
	Alphabetical bool `json:"alphabetical"`
}

// DefaultPreferences are what a first run starts with.
func DefaultPreferences() Preferences {
	return Preferences{FontSize: DefaultFontSize, Font: "go", ChapterTitles: true}
}

// Normalize puts every value in its range (a file edited by hand, or an older one).
func (p Preferences) Normalize() Preferences {
	if p.FontSize < MinFontSize || p.FontSize > MaxFontSize {
		p.FontSize = DefaultFontSize
	}
	p.FontSize -= (p.FontSize - MinFontSize) % FontSizeStep
	switch p.Font {
	case "go", "medium", "mono":
	default:
		p.Font = "go"
	}
	return p
}
