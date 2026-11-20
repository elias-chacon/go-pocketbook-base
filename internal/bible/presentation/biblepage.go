package presentation

import (
	"fmt"
	"strings"

	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// languageNames are the names of the languages of the versions in themselves.
var languageNames = map[string]string{
	"ar": "العربية", "cs": "Čeština", "da": "Dansk", "de": "Deutsch", "el": "Ελληνικά", "en": "English",
	"eo": "Esperanto", "es": "Español", "fa": "فارسی", "fi": "Suomi", "fr": "Français", "gr": "Ελληνικά",
	"he": "עברית", "hi": "हिन्दी", "hu": "Magyar", "id": "Indonesia", "it": "Italiano", "ja": "日本語",
	"ko": "한국어", "la": "Latina", "nb": "Norsk", "nl": "Nederlands", "pl": "Polski", "pt-br": "Português (Brasil)",
	"ro": "Română", "ru": "Русский", "sv": "Svenska", "sw": "Kiswahili", "tr": "Türkçe", "uk": "Українська",
	"vi": "Tiếng Việt", "zh": "中文",
}

// languageName is the name shown for a language tag.
func languageName(tag string) string {
	if n := languageNames[tag]; n != "" {
		return n
	}
	return strings.ToUpper(tag)
}

// biblePage is the second page: the bar of versions and search on top, and below it the
// books, the chapters of a book, the verses of a chapter, the reader or the search results.
func (s *Screen) biblePage() widget.Widget {
	page := widget.NewColumn(16)
	page.ID = "bible"
	if s.view == viewReader {
		s.buildReader(page)
		return page
	}
	page.AppendChild(s.versionBar())
	s.extra = 0
	if cont := s.continueButton(); cont != nil && s.view != viewSearch {
		page.AppendChild(cont)
		s.extra += widget.ControlHeight + 40
	}
	if s.Busy() {
		s.extra += jobStripHeight
	}
	switch {
	case len(s.versions) == 0:
		page.AppendChild(s.noVersions())
	case s.view == viewSearch:
		page.AppendChild(s.searchResults())
	case s.view == viewChapters:
		page.AppendChild(s.chaptersList())
	case s.view == viewVerses:
		page.AppendChild(s.versesGrid())
	default:
		page.AppendChild(s.booksList())
	}
	return page
}

// versionBar is the top bar: the version in use, the language and the version to pick and the
// search field.
func (s *Screen) versionBar() widget.Widget {
	current := s.t("bible.version.none")
	if v, ok := s.currentVersion(); ok {
		current = v.Label()
	}
	title := widget.NewLabel(s.t("bible.version.current", current))
	title.ID = "version-current"
	title.Style.Bold = true

	lang := s.activeLanguage()
	langBox := widget.NewSearchComboBox(s.t("bible.lang"))
	langBox.ID = "combo-language"
	s.searchable(langBox, s.t("bible.lang"))
	for _, l := range domain.LanguagesOf(s.versions) {
		langBox.Options = append(langBox.Options, widget.Option{Label: languageName(l.Tag), Value: l.Tag})
	}
	langBox.SetValue(lang)
	langBox.Style.FlexGrow = 1
	langBox.OnChange = func(*widget.Event) { s.pickLanguage(langBox.Value()) }

	versionBox := widget.NewSearchComboBox(s.t("bible.version.pick"))
	versionBox.ID = "combo-version"
	s.searchable(versionBox, s.t("bible.version.pick"))
	for _, v := range domain.VersionsIn(s.versions, lang) {
		state := s.t("bible.version.remote")
		if v.Downloaded {
			state = s.t("bible.version.downloaded")
		}
		versionBox.Options = append(versionBox.Options, widget.Option{Label: v.Label() + " · " + state, Value: v.Slug})
	}
	versionBox.SetValue(s.prefs.Version)
	versionBox.Style.FlexGrow = 2
	versionBox.OnChange = func(*widget.Event) { s.pickVersion(versionBox.Value()) }

	search := widget.NewSearchInput(s.t("bible.search.placeholder"))
	search.ID = "search"
	search.Value = s.query
	search.OnChange = func(*widget.Event) { s.Search(search.Value) }
	search.OnSubmit = search.OnChange

	row := widget.NewRow(12, langBox, versionBox)
	return widget.NewColumn(12, title, row, search)
}

// searchable gives a search combo box the texts of the app.
func (s *Screen) searchable(box *widget.SearchComboBox, title string) {
	box.Title = title
	box.SearchPlaceholder = s.t("bible.combo.search")
	box.CancelText = s.t("bible.combo.close")
	box.NoMatchText = s.t("bible.combo.none")
}

// activeLanguage is the language of the versions the picker shows: the one chosen, or the one
// of the version in use.
func (s *Screen) activeLanguage() string {
	if s.pickLang != "" {
		return s.pickLang
	}
	if v, ok := s.currentVersion(); ok {
		return v.Language
	}
	if s.prefs.VersionLang != "" {
		return strings.ToLower(s.prefs.VersionLang)
	}
	return strings.ToLower(s.tr.Lang())
}

// pickLanguage changes the language of the versions on offer (the version in use stays until another is chosen).
func (s *Screen) pickLanguage(tag string) {
	s.pickLang = tag
	s.rebuildBible()
}

// pickVersion makes a version the one in use, downloading it first when it is not on the reader.
func (s *Screen) pickVersion(slug string) {
	for _, v := range s.versions {
		if v.Slug != slug {
			continue
		}
		if v.Downloaded {
			s.useVersion(v)
			return
		}
		s.askDownload(v)
		return
	}
}

// useVersion switches the reading to a version that is on the reader, keeps the choice and shows
// the Bible page.
func (s *Screen) useVersion(v domain.Version) {
	s.adoptVersion(v)
	s.rebuildBible()
}

// adoptVersion makes a version that is on the reader the one in use and keeps the choice.
func (s *Screen) adoptVersion(v domain.Version) {
	s.prefs.Version, s.prefs.VersionLang = v.Slug, v.Language
	s.pickLang = ""
	if err := s.cfg.Bible.SavePreferences(s.prefs); err != nil {
		s.fail(err)
	}
	s.query, s.hits = "", nil
	if s.view == viewSearch {
		s.view = viewBooks
	}
}

// noVersions is the page before the list of versions was ever fetched.
func (s *Screen) noVersions() widget.Widget {
	get := widget.NewButton(s.t("bible.refresh"))
	get.ID = "refresh-versions"
	get.SetVariant(widget.VariantPrimary)
	get.OnClick = func(*widget.Event) { s.RefreshVersions() }
	return widget.NewColumn(16, widget.NewLabel(s.t("bible.versions.none")), get)
}

// rowsPerPage is how many list rows fit below the bar of the Bible page.
func (s *Screen) rowsPerPage(row float64) int {
	n := int((s.Doc.H - 820 - s.extra) / (row + 8))
	return int(geom.Clamp(float64(n), 4, 14))
}

// listButton is a full-width row of a list.
func listButton(text string, onClick func()) *widget.Button {
	b := widget.NewButton(text)
	b.Style.TextAlign = gfx.AlignStart
	b.Style.Bold = false
	b.Style.Padding = geom.XY(0, 24)
	b.OnClick = func(*widget.Event) { onClick() }
	return b
}

// booksList is the list of books, in the original order or alphabetical.
func (s *Screen) booksList() widget.Widget {
	books := domain.InCanonicalOrder(s.books)
	toggle := widget.NewButton(s.t("bible.books.alpha"))
	if s.prefs.Alphabetical {
		books = domain.SortAlphabetically(s.books, s.nameLang())
		toggle.Text = s.t("bible.books.canon")
	}
	toggle.ID = "books-order"
	toggle.OnClick = func(*widget.Event) { s.toggleOrder() }

	title := heading(s.t("bible.books"))
	title.Style.FlexGrow = 1
	head := widget.NewRow(12, title, toggle)
	head.Style.AlignItems = widget.AlignCenter

	items := make([]widget.Widget, len(books))
	for i, b := range books {
		b := b
		row := listButton(s.bookName(b), func() { s.OpenBook(b.ID) })
		row.ID = fmt.Sprintf("book-%d", b.ID)
		items[i] = row
	}
	pager := widget.NewPager(s.rowsPerPage(widget.ControlHeight), items...)
	pager.GoTo(s.bookPage)
	pager.OnPage = func(*widget.Event) { s.bookPage = pager.Page() }
	return widget.NewColumn(12, head, pager.Root)
}

// nameLang is the language book names are sorted in.
func (s *Screen) nameLang() string {
	for _, lang := range []string{strings.ToLower(s.prefs.VersionLang), strings.ToLower(s.tr.Lang())} {
		for _, b := range s.books {
			if b.Names[lang] != "" {
				return lang
			}
		}
	}
	return "en"
}

// toggleOrder switches between the original and the alphabetical order of the books.
func (s *Screen) toggleOrder() {
	s.prefs.Alphabetical = !s.prefs.Alphabetical
	s.bookPage = 0
	if err := s.cfg.Bible.SavePreferences(s.prefs); err != nil {
		s.fail(err)
	}
	s.rebuildBible()
}

// OpenBook shows the chapters of a book.
func (s *Screen) OpenBook(id int) {
	s.view, s.book, s.listPage = viewChapters, id, 0
	s.rebuildBible()
}

// backBar is the line at the top of a level: a button to go up and the title.
func (s *Screen) backBar(backText, title string, back func()) widget.Widget {
	b := widget.NewButton(backText)
	b.ID = "back"
	b.Icon = gfx.IconChevronLeft
	b.OnClick = func(*widget.Event) { back() }
	t := heading(title)
	t.Style.FlexGrow = 1
	row := widget.NewRow(16, b, t)
	row.Style.AlignItems = widget.AlignCenter
	return row
}

// chaptersList is the list of chapters of the open book: a number each, or the number and the
// start of the chapter when the preferences ask for it and the version is on the reader.
func (s *Screen) chaptersList() widget.Widget {
	b, ok := s.bookByID(s.book)
	if !ok {
		s.view = viewBooks
		return s.booksList()
	}
	bar := s.backBar(s.t("bible.back.books"), s.t("bible.chapters.of", s.bookName(b)), func() { s.view = viewBooks; s.rebuildBible() })

	var previews map[int]string
	if s.prefs.ChapterTitles && s.ready() {
		var err error
		if previews, err = s.cfg.Bible.ChapterPreviews(s.prefs.Version, b.ID); err != nil {
			s.fail(err)
		}
	}
	col := widget.NewColumn(12, bar)
	if !s.ready() {
		col.AppendChild(s.needsDownload())
	}
	if len(previews) > 0 {
		items := make([]widget.Widget, b.Chapters)
		for ch := 1; ch <= b.Chapters; ch++ {
			ch := ch
			text := s.t("bible.chapter", ch)
			if p := strings.TrimSpace(previews[ch]); p != "" {
				text = fmt.Sprintf("%d · %s", ch, p)
			}
			row := listButton(text, func() { s.OpenChapter(b.ID, ch) })
			row.ID = fmt.Sprintf("chapter-%d", ch)
			items[ch-1] = row
		}
		pager := widget.NewPager(s.rowsPerPage(widget.ControlHeight)-1, items...)
		pager.GoTo(s.listPage)
		pager.OnPage = func(*widget.Event) { s.listPage = pager.Page() }
		col.AppendChild(pager.Root)
		return col
	}
	col.AppendChild(s.numberGrid(b.Chapters, "chapter", func(n int) { s.OpenChapter(b.ID, n) }))
	return col
}

// OpenChapter shows the verses of a chapter, for jumping to one.
func (s *Screen) OpenChapter(book, chapter int) {
	s.view, s.book, s.chapter, s.listPage = viewVerses, book, chapter, 0
	s.rebuildBible()
}

// versesGrid is the grid of verse numbers of the open chapter, and a button to read it from the start.
func (s *Screen) versesGrid() widget.Widget {
	b, ok := s.bookByID(s.book)
	if !ok {
		s.view = viewBooks
		return s.booksList()
	}
	title := s.t("bible.verses.of", s.bookName(b), s.chapter)
	bar := s.backBar(s.bookName(b), title, func() { s.view = viewChapters; s.listPage = 0; s.rebuildBible() })
	col := widget.NewColumn(12, bar)
	if !s.ready() {
		col.AppendChild(s.needsDownload())
		return col
	}
	text, err := s.cfg.Bible.Chapter(s.prefs.Version, s.book, s.chapter)
	if err != nil {
		s.fail(err)
	}
	if len(text.Verses) == 0 {
		col.AppendChild(widget.NewLabel(s.t("bible.nochapter")))
		return col
	}
	read := widget.NewButton(s.t("bible.read.chapter"))
	read.ID = "read-chapter"
	read.SetVariant(widget.VariantPrimary)
	read.OnClick = func(*widget.Event) { s.OpenReader(s.book, s.chapter, text.Verses[0].Number) }
	col.AppendChild(read)
	col.AppendChild(muted(s.t("bible.verse.hint")))
	numbers := make([]int, len(text.Verses))
	for i, v := range text.Verses {
		numbers[i] = v.Number
	}
	col.AppendChild(s.pickGrid(numbers, "verse", func(n int) { s.OpenReader(s.book, s.chapter, n) }))
	return col
}

// gridColumns is how many numbers a row of a number grid has.
const gridColumns = 6

// numberGrid is a pager of buttons numbered 1..count.
func (s *Screen) numberGrid(count int, idPrefix string, pick func(int)) widget.Widget {
	numbers := make([]int, count)
	for i := range numbers {
		numbers[i] = i + 1
	}
	return s.pickGrid(numbers, idPrefix, pick)
}

// pickGrid is a pager of rows of numbered buttons.
func (s *Screen) pickGrid(numbers []int, idPrefix string, pick func(int)) widget.Widget {
	var rows []widget.Widget
	for start := 0; start < len(numbers); start += gridColumns {
		row := widget.NewRow(10)
		for i := 0; i < gridColumns; i++ {
			if start+i >= len(numbers) {
				filler := widget.NewLabel("")
				filler.Style.FlexGrow = 1
				row.AppendChild(filler)
				continue
			}
			n := numbers[start+i]
			b := widget.NewButton(fmt.Sprint(n))
			b.ID = fmt.Sprintf("%s-%d", idPrefix, n)
			b.Style.FlexGrow, b.Style.Padding = 1, geom.XY(0, 12)
			b.OnClick = func(*widget.Event) { pick(n) }
			row.AppendChild(b)
		}
		rows = append(rows, row)
	}
	perPage := s.rowsPerPage(widget.ControlHeight) - 2
	pager := widget.NewPager(perPage, rows...)
	pager.GoTo(s.listPage)
	pager.OnPage = func(*widget.Event) { s.listPage = pager.Page() }
	return pager.Root
}

// needsDownload is the notice and button shown where the text of the version is needed.
func (s *Screen) needsDownload() widget.Widget {
	col := widget.NewColumn(12, widget.NewLabel(s.t("bible.notext")))
	if v, ok := s.currentVersion(); ok {
		b := widget.NewButton(s.t("bible.download") + " " + v.Label())
		b.ID = "download-current"
		b.SetVariant(widget.VariantPrimary)
		b.OnClick = func(*widget.Event) { s.askDownload(v) }
		col.AppendChild(b)
	}
	return col
}

// Search looks for the words in the text of the version in use and shows the results. An empty
// query goes back to the list of books.
func (s *Screen) Search(query string) {
	s.query = strings.TrimSpace(query)
	if s.query == "" {
		s.hits = nil
		s.view = viewBooks
		s.rebuildBible()
		return
	}
	s.view, s.listPage = viewSearch, 0
	s.hits = nil
	if s.ready() {
		hits, err := s.cfg.Bible.Search(s.prefs.Version, s.query)
		if err != nil {
			s.fail(err)
		}
		s.hits = hits
	}
	s.rebuildBible()
}

// searchResults is the list of verses found.
func (s *Screen) searchResults() widget.Widget {
	clear := widget.NewButton(s.t("bible.search.clear"))
	clear.ID = "search-clear"
	clear.OnClick = func(*widget.Event) { s.Search("") }
	col := widget.NewColumn(12)
	switch {
	case !s.ready():
		col.AppendChild(s.needsDownload())
	case len(s.hits) == 0:
		col.AppendChild(widget.NewLabel(s.t("bible.search.none", s.query)))
	default:
		col.AppendChild(muted(s.t("bible.search.results", len(s.hits), s.query)))
		items := make([]widget.Widget, len(s.hits))
		for i, h := range s.hits {
			h := h
			text := s.refText(h.Ref) + "  " + oneLine(h.Text, 90)
			row := listButton(text, func() { s.OpenReader(h.Ref.Book, h.Ref.Chapter, h.Ref.Verse) })
			row.ID = fmt.Sprintf("hit-%d", i)
			items[i] = row
		}
		pager := widget.NewPager(s.rowsPerPage(widget.ControlHeight)-1, items...)
		pager.GoTo(s.listPage)
		pager.OnPage = func(*widget.Event) { s.listPage = pager.Page() }
		col.AppendChild(pager.Root)
	}
	col.AppendChild(clear)
	return col
}

// oneLine shortens a text to a line of at most n letters.
func oneLine(text string, n int) string {
	r := []rune(strings.Join(strings.Fields(text), " "))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n-1]) + "…"
}
