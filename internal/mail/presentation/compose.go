package presentation

import (
	"strings"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// The day and time of a quote's attribution, and of a forward's header.
const quoteDate = "02/01/2006 15:04"

// composeRows is how many lines the body field shows.
const composeRows = 14

// composer is the "new message" form.
type composer struct {
	draft       domain.Draft
	to, cc, bcc *widget.Input
	subject     *widget.Input
	body        *widget.TextArea
	note        *widget.Label
}

// Composing reports whether the new message form is showing.
func (s *Screen) Composing() bool { return s.form != nil }

// NewMessage opens an empty message.
func (s *Screen) NewMessage() { s.Compose(s.cfg.Mailbox.New()) }

// Reply opens the answer to the message that is open.
func (s *Screen) Reply(all bool) {
	if s.reading == nil {
		return
	}
	m := *s.reading
	d, err := s.cfg.Mailbox.Reply(m.ID, all, s.t("mail.wrote", m.Date.Local().Format(quoteDate), sender(s, m)))
	if err != nil {
		s.say(s.t(keyOpenFailed, err.Error()))
		return
	}
	s.Compose(d)
}

// Forward opens a forward of the message that is open.
func (s *Screen) Forward() {
	if s.reading == nil {
		return
	}
	m := *s.reading
	d, err := s.cfg.Mailbox.Forward(m.ID, s.t("mail.forwarded", sender(s, m), m.Date.Local().Format(quoteDate), m.Subject))
	if err != nil {
		s.say(s.t(keyOpenFailed, err.Error()))
		return
	}
	s.Compose(d)
}

func (s *Screen) field(id, placeholder, value string) *widget.Input {
	in := widget.NewInput(placeholder)
	in.ID, in.Value = id, value
	in.Style.FlexGrow = 1
	return in
}

func joinAddresses(list []domain.Address) string {
	parts := make([]string, len(list))
	for i, a := range list {
		parts[i] = a.String()
	}
	return strings.Join(parts, ", ")
}

// Compose shows the form for a draft.
func (s *Screen) Compose(d domain.Draft) {
	f := &composer{draft: d}
	s.form = f
	f.to = s.field("mail-to", s.t("mail.to"), joinAddresses(d.To))
	f.cc = s.field("mail-cc", s.t("mail.cc"), joinAddresses(d.Cc))
	f.bcc = s.field("mail-bcc", s.t("mail.bcc"), joinAddresses(d.Bcc))
	f.subject = s.field("mail-subject", s.t("mail.subject"), d.Subject)
	f.body = widget.NewTextArea(s.t("mail.body"), composeRows)
	f.body.ID, f.body.Value = "mail-body-text", d.Body
	f.body.Style.Width = widget.Pct(100)
	f.note = widget.NewLabel(s.cfg.Notice)
	f.note.Style.FontSize, f.note.Style.Color = widget.FontSmall, widget.Muted

	title := widget.NewLabel(s.t("mail.compose"))
	title.Style.FontSize, title.Style.Bold, title.Style.FlexGrow = widget.FontTitle, true, 1
	send := widget.NewButton(s.t("mail.send"))
	send.ID = "mail-send"
	send.OnClick = func(*widget.Event) { s.SendDraft() }
	cancel := widget.NewButton(s.t("mail.cancel"))
	cancel.ID = "mail-cancel"
	cancel.OnClick = func(*widget.Event) { s.CancelCompose() }
	save := widget.NewButton(s.t("mail.savedraft"))
	save.ID = "mail-savedraft"
	save.OnClick = func(*widget.Event) { s.SaveDraft() }
	bar := widget.NewRow(12, title, cancel, save, send)
	bar.Style.Padding = geom.XY(28, 28)
	bar.Style.AlignItems = widget.AlignCenter
	paper(bar)

	form := widget.NewColumn(16, f.to, f.cc, f.bcc, f.subject, f.body, f.note)
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	s.root = widget.NewColumn(0, bar, form)
	s.root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.root)
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

// CancelCompose throws the message away and goes back to the list.
func (s *Screen) CancelCompose() {
	s.form = nil
	s.showList()
}

// read copies the form into a draft; an address that cannot be read is an error.
func (f *composer) read() (domain.Draft, error) {
	d := f.draft
	var err error
	lists := []struct {
		field *widget.Input
		into  *[]domain.Address
	}{{f.to, &d.To}, {f.cc, &d.Cc}, {f.bcc, &d.Bcc}}
	for _, l := range lists {
		if *l.into, err = domain.ParseAddressList(l.field.Value); err != nil {
			return d, err
		}
	}
	d.Subject, d.Body = f.subject.Value, f.body.Value
	return d, nil
}

// SaveDraft keeps the message for later and goes back to the list.
func (s *Screen) SaveDraft() {
	f := s.form
	if f == nil {
		return
	}
	d, err := f.read()
	if err != nil {
		f.note.Text = s.t("mail.badaddress", err.Error())
		s.Doc.Invalidate()
		return
	}
	if _, err := s.cfg.Mailbox.SaveDraft(d); err != nil {
		f.note.Text = s.t("mail.draft.failed", err.Error())
		s.Doc.Invalidate()
		return
	}
	s.form = nil
	s.cfg.Notice = s.t("mail.draft.saved")
	s.showList()
}

// ShowDrafts lists the saved drafts: tapping one opens it to go on writing.
func (s *Screen) ShowDrafts() {
	drafts, err := s.cfg.Mailbox.DraftList()
	if err != nil {
		s.say(s.t("mail.draft.failed", err.Error()))
		return
	}
	back := widget.NewButton(s.t("mail.back"))
	back.ID = "drafts-back"
	back.OnClick = func(*widget.Event) { s.showList() }
	title := widget.NewLabel(s.t("mail.drafts"))
	title.Style.FontSize, title.Style.Bold, title.Style.FlexGrow = widget.FontTitle, true, 1
	bar := widget.NewRow(12, title, back)
	bar.Style.Padding = geom.XY(28, 28)
	bar.Style.AlignItems = widget.AlignCenter
	paper(bar)

	body := widget.NewColumn(8)
	body.Style.Padding = geom.XY(8, 28)
	body.Style.FlexGrow = 1
	if len(drafts) == 0 {
		empty := widget.NewLabel(s.t("mail.drafts.empty"))
		empty.Style.Color, empty.Style.Padding = widget.Muted, geom.All(28)
		body.AppendChild(empty)
	}
	for i, d := range drafts {
		if i >= rowsPerPage {
			break
		}
		d := d
		who := joinAddresses(d.To)
		if who == "" {
			who = s.t("mail.nobody")
		}
		to := widget.NewLabel(who)
		to.Style.Bold = true
		subject := widget.NewLabel(orElse(d.Subject, s.t("mail.nosubject")))
		row := widget.NewColumn(2, to, subject)
		row.ID = "draft-" + d.ID
		row.Style.Padding = geom.XY(12, 20)
		row.Style.BorderColor, row.Style.BorderWidth, row.Style.BorderRadius = widget.Mid, 2, widget.Radius
		paper(row)
		row.OnClick = func(*widget.Event) { s.Compose(d) }
		body.AppendChild(row)
	}
	s.root = widget.NewColumn(0, bar, body)
	s.root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.root)
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

// SendDraft sends what was written. A message that does not go stays on the screen.
func (s *Screen) SendDraft() {
	f := s.form
	if f == nil {
		return
	}
	d, err := f.read()
	if err != nil {
		f.note.Text = s.t("mail.badaddress", err.Error())
		s.Doc.Invalidate()
		return
	}
	f.note.Text = s.t("mail.sending")
	s.Doc.Invalidate()
	if s.cfg.Flush != nil {
		s.cfg.Flush()
	}
	if err := s.cfg.Mailbox.Send(d); err != nil {
		f.note.Text = s.t("mail.send.failed", err.Error())
		s.Doc.Invalidate()
		return
	}
	s.form = nil
	s.cfg.Notice = s.t("mail.sent")
	s.showList()
}
