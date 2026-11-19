// Package presentation draws the Mail app: the folder list, the messages of a folder a
// page at a time, and a message read page by page. It is built only from the
// toolkit's widgets and holds no rules of its own: it reads the cache and asks the
// Mailbox use case to change things.
package presentation

import (
	"embed"
	"errors"
	"strconv"
	"strings"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/mail/application"
	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads the Mail app's texts (pt-BR first, English fallback) in lang.
func NewTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// Mailbox is what the screen needs of the Mail context (application.Mailbox).
type Mailbox interface {
	Folders() ([]domain.Folder, error)
	Page(folderID string, offset, limit int) ([]domain.Message, int, error)
	Find(text string, limit int) ([]domain.Message, error)
	Run() (application.Result, error)
	Open(id string) (domain.Message, error)
	New() domain.Draft
	Reply(id string, all bool, attribution string) (domain.Draft, error)
	Forward(id, header string) (domain.Draft, error)
	Send(d domain.Draft) error
	SaveDraft(d domain.Draft) (domain.Draft, error)
	DraftList() ([]domain.Draft, error)
	DeleteDraft(id string) error
	MarkRead(id string, read bool) error
	SetFlagged(id string, flagged bool) error
	Delete(id string) error
}

// Config is everything the screen needs.
type Config struct {
	Width, Height int
	// Doc is the document to draw on (a new one when nil): an app has one document for
	// its whole life, so the screens it switches between share it.
	Doc *widget.Document
	// Chrome is the standard top bar (the app name with About and Exit), above every screen.
	Chrome   widget.AppChrome
	Keyboard widget.Keyboard
	Tr       *i18n.Bundle
	Mailbox  Mailbox
	// Flush paints pending changes now, before a slow call (nil: no flush).
	Flush func()
	// Now is the clock (time.Now when nil); the dates of the list depend on it.
	Now func() time.Time
	// Signature is added to the end of every new message.
	Signature string
	// OnAccount opens the account settings (nil: no Account button).
	OnAccount func()
	// Notice is shown at the first paint (a message from the composition root).
	Notice string
}

// keyOpenFailed is the text for a message that could not be read.
const (
	keyOpenFailed = "mail.open.failed"
	keyMarkFailed = "mail.mark.failed"
)

const (
	rowsPerPage = 7
	findLimit   = 100
	bodyLines   = 22
)

// Screen is the Mail app's screen: the list of a folder, or a message.
type Screen struct {
	Doc *widget.Document

	cfg     Config
	folders []domain.Folder
	folder  domain.Folder
	query   string
	page    int
	items   []domain.Message // the rows of the page
	total   int              // rows in all (for the pages)
	found   []domain.Message // the results of a search, all of them
	reading *domain.Message  // non-nil while a message is open
	form    *composer        // non-nil while a message is being written

	root            *widget.Panel
	search          *widget.Input
	body            *widget.Panel
	note, indicator *widget.Label
	prev, next      *widget.Button
}

// NewScreen builds the screen and shows the first folder.
func NewScreen(cfg Config) *Screen {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	s := &Screen{cfg: cfg}
	s.Doc = cfg.Doc
	if s.Doc == nil {
		s.Doc = widget.NewDocument(cfg.Width, cfg.Height)
		s.Doc.Keyboard = cfg.Keyboard
	}
	s.loadFolders()
	s.showList()
	if cfg.Notice != "" {
		s.say(cfg.Notice)
	}
	return s
}

func (s *Screen) t(key string, args ...interface{}) string { return s.cfg.Tr.T(key, args...) }

// Folder is the folder being shown.
func (s *Screen) Folder() domain.Folder { return s.folder }

// Reading is the open message (nil when the list is showing).
func (s *Screen) Reading() *domain.Message { return s.reading }

func (s *Screen) loadFolders() {
	folders, err := s.cfg.Mailbox.Folders()
	if err != nil {
		s.cfg.Notice = s.t("mail.sync.failed", err.Error())
	}
	s.folders = folders
	if s.folder.ID != "" {
		for _, f := range folders {
			if f.ID == s.folder.ID {
				s.folder = f
				return
			}
		}
	}
	s.folder, _ = application.Inbox(folders)
}

// folderName is what the folder is called on screen.
func (s *Screen) folderName(f domain.Folder) string {
	if f.Role != domain.Other && f.Role != "" {
		if key := "role." + string(f.Role); s.cfg.Tr.Has(key) {
			return s.t(key)
		}
	}
	return f.Name
}

// say puts a message in the footer.
func (s *Screen) say(text string) {
	s.cfg.Notice = text
	if s.note != nil {
		s.note.Text = text
		s.Doc.Invalidate()
	}
}

func paper(w widget.Widget) widget.Widget {
	w.Elem().Style.Background = widget.Paper
	return w
}

// ----- the list -----

func (s *Screen) title() string {
	name := s.t("mail.title")
	if s.folder.ID != "" {
		name = s.folderName(s.folder)
	}
	if s.folder.Unread > 0 {
		name += "  ·  " + s.t("mail.unread", s.folder.Unread)
	}
	return name
}

func (s *Screen) showList() {
	s.reading = nil
	title := widget.NewLabel(s.title())
	title.ID = "mail-title"
	title.Style.FontSize, title.Style.Bold, title.Style.FlexGrow = widget.FontTitle, true, 1

	items := make([]widget.MenuItem, 0, len(s.folders))
	for _, f := range s.folders {
		f := f
		label := s.folderName(f)
		if f.Unread > 0 {
			label += " (" + strconv.Itoa(f.Unread) + ")"
		}
		items = append(items, widget.MenuItem{Label: label, Value: f.ID, OnSelect: func() { s.OpenFolder(f.ID) }})
	}
	folders := widget.NewDropdown(s.t("mail.folders"), items...)
	folders.ID = "mail-folders"
	sync := widget.NewButton(s.t("mail.sync"))
	sync.ID = "mail-sync"
	sync.OnClick = func(*widget.Event) { s.Sync() }
	compose := widget.NewButton(s.t("mail.new"))
	compose.ID = "mail-new"
	compose.OnClick = func(*widget.Event) { s.NewMessage() }
	drafts := widget.NewButton(s.t("mail.drafts"))
	drafts.ID = "mail-drafts"
	drafts.OnClick = func(*widget.Event) { s.ShowDrafts() }
	head := widget.NewRow(12, title, folders, sync, compose)
	head.Style.Padding = geom.XY(28, 28)
	head.Style.AlignItems = widget.AlignCenter
	paper(head)

	s.search = widget.NewSearchInput(s.t("mail.search"))
	s.search.ID = "mail-search"
	s.search.Value = s.query
	s.search.Style.FlexGrow = 1
	s.search.OnSubmit = func(*widget.Event) { s.Search(s.search.Value) }
	s.search.OnChange = s.search.OnSubmit
	bar := widget.NewRow(12, s.search, drafts)
	if s.cfg.OnAccount != nil {
		account := widget.NewButton(s.t("mail.account"))
		account.ID = "mail-account"
		account.OnClick = func(*widget.Event) { s.cfg.OnAccount() }
		bar.AppendChild(account)
	}
	bar.Style.Padding = geom.XY(0, 28)

	s.body = widget.NewColumn(8)
	s.body.ID = "mail-body"
	s.body.Style.FlexGrow, s.body.Style.Padding = 1, geom.XY(12, 28)

	s.note = widget.NewLabel(s.cfg.Notice)
	s.note.Style.FontSize, s.note.Style.Color, s.note.Style.FlexGrow = widget.FontSmall, widget.Muted, 1
	s.indicator = widget.NewLabel("")
	s.indicator.Style.FontSize, s.indicator.Style.Bold = widget.FontSmall, true
	s.indicator.Style.TextAlign, s.indicator.Style.Width = gfx.AlignCenter, widget.Px(110)
	s.prev, s.next = widget.NewIconButton(gfx.IconChevronLeft), widget.NewIconButton(gfx.IconChevronRight)
	s.prev.ID, s.next.ID = "mail-prev", "mail-next"
	s.prev.OnClick = func(*widget.Event) { s.GoTo(s.page - 1) }
	s.next.OnClick = func(*widget.Event) { s.GoTo(s.page + 1) }
	foot := widget.NewRow(12, s.note, s.prev, s.indicator, s.next)
	foot.Style.Padding = geom.XY(14, 28)
	paper(foot)

	s.root = widget.NewColumn(0, head, bar, s.body, foot)
	s.root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.cfg.Chrome.Wrap(s.Doc, s.root))
	s.reload()
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

// OpenFolder shows another folder, from its first page, and ends a search.
func (s *Screen) OpenFolder(id string) {
	for _, f := range s.folders {
		if f.ID == id {
			s.folder, s.query, s.page = f, "", 0
			s.showList()
			return
		}
	}
}

// Search lists the messages that contain the text (an empty text goes back to the folder).
func (s *Screen) Search(text string) {
	text = strings.TrimSpace(text)
	if text == s.query {
		return
	}
	s.query, s.page = text, 0
	s.reload()
	s.Doc.Invalidate()
}

// GoTo shows a page of the list (clamped).
func (s *Screen) GoTo(page int) {
	if pages := s.pages(); page >= pages {
		page = pages - 1
	}
	if page < 0 {
		page = 0
	}
	if page == s.page {
		return
	}
	s.page = page
	s.reload()
	s.Doc.Invalidate()
}

// Page is the page of the list, counted from 0.
func (s *Screen) Page() int { return s.page }

func (s *Screen) pages() int {
	if s.total <= 0 {
		return 1
	}
	return (s.total + rowsPerPage - 1) / rowsPerPage
}

// reload reads the rows of the current page from the cache and rebuilds the body.
func (s *Screen) reload() {
	var err error
	if s.query != "" {
		if s.found, err = s.cfg.Mailbox.Find(s.query, findLimit); err == nil {
			s.total = len(s.found)
			from, to := s.page*rowsPerPage, (s.page+1)*rowsPerPage
			if from > len(s.found) {
				from = len(s.found)
			}
			if to > len(s.found) {
				to = len(s.found)
			}
			s.items = s.found[from:to]
		}
	} else if s.folder.ID != "" {
		s.items, s.total, err = s.cfg.Mailbox.Page(s.folder.ID, s.page*rowsPerPage, rowsPerPage)
	} else {
		s.items, s.total = nil, 0
	}
	if err != nil {
		s.items, s.total = nil, 0
		s.cfg.Notice = s.t(keyOpenFailed, err.Error())
		s.note.Text = s.cfg.Notice
	}
	s.fillBody()
}

func (s *Screen) fillBody() {
	s.body.ClearChildren()
	if len(s.items) == 0 {
		text := s.t("mail.empty")
		if s.query != "" {
			text = s.t("mail.nomatch", s.query)
		}
		empty := widget.NewLabel(text)
		empty.Style.Color, empty.Style.Padding = widget.Muted, geom.All(28)
		s.body.AppendChild(empty)
	}
	for _, m := range s.items {
		s.body.AppendChild(s.row(m))
	}
	s.indicator.Text = s.t("mail.page", s.page+1, s.pages())
	s.prev.Disabled = s.page == 0
	s.next.Disabled = s.page >= s.pages()-1
}

func (s *Screen) row(m domain.Message) widget.Widget {
	from := widget.NewLabel(sender(s, m))
	from.Style.Bold, from.Style.FlexGrow = m.Unread(), 1
	date := widget.NewLabel(shortDate(m.Date, s.cfg.Now(), s.t("yesterday")))
	date.Style.FontSize, date.Style.Color = widget.FontSmall, widget.Muted
	name := orElse(m.Subject, s.t("mail.nosubject"))
	if m.Has(domain.Flagged) {
		name = "! " + name
	}
	subject := widget.NewLabel(name)
	subject.Style.Bold = m.Unread() || m.Has(domain.Flagged)
	preview := widget.NewLabel(m.Preview)
	preview.Style.FontSize, preview.Style.Color = widget.FontSmall, widget.Muted
	row := widget.NewColumn(2, widget.NewRow(8, from, date), subject, preview)
	row.ID = "msg-" + m.ID
	row.Style.Padding = geom.XY(12, 20)
	row.Style.BorderColor, row.Style.BorderWidth, row.Style.BorderRadius = widget.Mid, 2, widget.Radius
	paper(row)
	id := m.ID
	row.OnClick = func(*widget.Event) { s.Open(id) }
	return row
}

func sender(s *Screen, m domain.Message) string {
	if m.From.Email == "" && m.From.Name == "" {
		return s.t("mail.nosender")
	}
	return m.From.Display()
}

func orElse(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// ----- synchronization -----

// Sync brings the newest messages from the server and refreshes the list.
func (s *Screen) Sync() {
	s.say(s.t("mail.syncing"))
	if s.cfg.Flush != nil {
		s.cfg.Flush()
	}
	res, err := s.cfg.Mailbox.Run()
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		s.say(s.t("mail.sync.signin"))
		return
	case err != nil:
		s.say(s.t("mail.sync.failed", err.Error()))
		return
	}
	s.loadFolders()
	s.page = 0
	s.showList()
	if len(res.Errors) > 0 {
		s.say(s.t("mail.synced.partial", res.Messages, len(res.Errors), res.Errors[0].Error()))
		return
	}
	s.say(s.t("mail.synced", res.Messages, res.Folders))
}

// ----- reading -----

// Open shows a message, fetching its text if it was not read before, and marks it read.
func (s *Screen) Open(id string) {
	m, err := s.cfg.Mailbox.Open(id)
	if err != nil && m.ID == "" {
		s.say(s.t(keyOpenFailed, err.Error()))
		return
	}
	if err != nil {
		s.say(s.t(keyOpenFailed, err.Error())) // the preview is still shown
	}
	if m.Unread() {
		if markErr := s.cfg.Mailbox.MarkRead(m.ID, true); markErr != nil {
			s.say(s.t(keyMarkFailed, markErr.Error()))
		}
		m.Flags |= domain.Seen
	}
	s.showMessage(m)
}

func (s *Screen) showMessage(m domain.Message) {
	s.reading = &m
	back := widget.NewButton(s.t("mail.back"))
	back.ID = "mail-back"
	back.OnClick = func(*widget.Event) { s.Close() }
	unread := widget.NewButton(s.t("mail.markunread"))
	unread.ID = "mail-unread"
	unread.OnClick = func(*widget.Event) { s.MarkUnread() }
	del := widget.NewButton(s.t("mail.delete"))
	del.ID = "mail-delete"
	del.OnClick = func(*widget.Event) { s.DeleteOpen() }
	spacer := widget.NewLabel("")
	spacer.Style.FlexGrow = 1
	flag := widget.NewButton(s.t(flagKey(m)))
	flag.ID = "mail-flag"
	flag.OnClick = func(*widget.Event) { s.ToggleFlag() }
	bar := widget.NewRow(12, back, spacer, flag, unread, del)
	bar.Style.Padding = geom.XY(28, 28)
	paper(bar)
	answers := s.answerBar()

	subject := widget.NewLabel(orElse(m.Subject, s.t("mail.nosubject")))
	subject.Style.FontSize, subject.Style.Bold = widget.FontTitle, true
	head := widget.NewColumn(4, subject, s.meta("mail.from", sender(s, m)), s.meta("mail.to", list(m.To, s.t("mail.nobody"))))
	if len(m.Cc) > 0 {
		head.AppendChild(s.meta("mail.cc", list(m.Cc, "")))
	}
	head.AppendChild(widget.NewLabel(m.Date.Local().Format("02/01/2006 15:04")))
	if len(m.Attachments) > 0 {
		names := make([]string, len(m.Attachments))
		for i, a := range m.Attachments {
			names[i] = a.Name
		}
		head.AppendChild(widget.NewLabel(s.t("mail.attachments", strings.Join(names, ", "))))
	}
	head.Style.Padding = geom.XY(14, 28)
	paper(head)

	text := widget.NewTextPager(orElse(m.Body, m.Preview))
	text.ID, text.Lines = "mail-text", bodyLines
	text.Style.FlexGrow, text.Style.Padding = 1, geom.XY(14, 28)

	s.note = widget.NewLabel(s.cfg.Notice)
	s.note.Style.FontSize, s.note.Style.Color = widget.FontSmall, widget.Muted
	s.note.Style.Padding = geom.XY(8, 28)
	s.root = widget.NewColumn(0, bar, answers, head, text, s.note)
	s.root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.cfg.Chrome.Wrap(s.Doc, s.root))
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

// answerBar has the buttons that start a message about the one being read.
func (s *Screen) answerBar() widget.Widget {
	mk := func(id, key string, fn func()) *widget.Button {
		b := widget.NewButton(s.t(key))
		b.ID = id
		b.OnClick = func(*widget.Event) { fn() }
		return b
	}
	bar := widget.NewRow(12,
		mk("mail-reply", "mail.reply", func() { s.Reply(false) }),
		mk("mail-replyall", "mail.replyall", func() { s.Reply(true) }),
		mk("mail-forward", "mail.forward", func() { s.Forward() }))
	bar.Style.Padding = geom.XY(28, 0)
	paper(bar)
	return bar
}

func flagKey(m domain.Message) string {
	if m.Has(domain.Flagged) {
		return "mail.unflag"
	}
	return "mail.flag"
}

// ToggleFlag flags the open message for follow-up, or removes the flag.
func (s *Screen) ToggleFlag() {
	if s.reading == nil {
		return
	}
	m := *s.reading
	on := !m.Has(domain.Flagged)
	if err := s.cfg.Mailbox.SetFlagged(m.ID, on); err != nil {
		s.say(s.t(keyMarkFailed, err.Error()))
	}
	if on {
		m.Flags |= domain.Flagged
	} else {
		m.Flags &^= domain.Flagged
	}
	s.showMessage(m)
}

func (s *Screen) meta(key, value string) widget.Widget {
	l := widget.NewLabel(s.t(key) + ": " + value)
	l.Style.FontSize = widget.FontSmall
	return l
}

func list(addrs []domain.Address, empty string) string {
	if len(addrs) == 0 {
		return empty
	}
	names := make([]string, len(addrs))
	for i, a := range addrs {
		names[i] = a.Display()
	}
	return strings.Join(names, ", ")
}

// Close goes back from a message to the list, where the unread counts are fresh.
func (s *Screen) Close() {
	s.loadFolders()
	s.showList()
}

// MarkUnread marks the open message unread and goes back to the list.
func (s *Screen) MarkUnread() {
	if s.reading == nil {
		return
	}
	if err := s.cfg.Mailbox.MarkRead(s.reading.ID, false); err != nil {
		s.say(s.t(keyMarkFailed, err.Error()))
	}
	s.Close()
}

// DeleteOpen deletes the open message and goes back to the list.
func (s *Screen) DeleteOpen() {
	if s.reading == nil {
		return
	}
	if err := s.cfg.Mailbox.Delete(s.reading.ID); err != nil {
		s.say(s.t("mail.delete.failed", err.Error()))
		return
	}
	s.cfg.Notice = s.t("mail.deleted")
	s.Close()
}

// Key handles the reader's page keys: they turn the list's pages, or the message's.
func (s *Screen) Key(previous bool) bool {
	if s.reading != nil {
		return false
	}
	if previous {
		s.GoTo(s.page - 1)
	} else {
		s.GoTo(s.page + 1)
	}
	return true
}
