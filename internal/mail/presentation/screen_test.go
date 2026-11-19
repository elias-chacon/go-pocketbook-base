package presentation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/mail/application"
	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

var clock = time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

// fakeBox is a Mailbox over a few messages in memory.
type fakeBox struct {
	folders []domain.Folder
	msgs    map[string][]domain.Message // by folder id, newest first
	byID    map[string]domain.Message

	syncs, marked, deleted int
	lastRead               bool
	syncErr, openErr       error
	markErr, deleteErr     error
	syncResult             application.Result
	replyErr, sendErr      error
	draftErr               error
	drafts                 []domain.Draft
	flagged                []bool
	sent                   []domain.Draft
}

func newBox(inbox int) *fakeBox {
	b := &fakeBox{
		folders: []domain.Folder{
			{ID: "INBOX", Name: "INBOX", Role: domain.Inbox, Unread: 2},
			{ID: "Sent", Name: "Sent", Role: domain.Sent},
			{ID: "Projetos", Name: "Projetos", Role: domain.Other},
		},
		msgs: map[string][]domain.Message{}, byID: map[string]domain.Message{},
	}
	for i := 0; i < inbox; i++ {
		m := domain.Message{
			ID: fmt.Sprintf("m%02d", i), FolderID: "INBOX", Subject: fmt.Sprintf("Assunto %d", i),
			From: domain.Address{Name: "Ana", Email: "ana@example.org"}, To: []domain.Address{{Email: "eu@example.org"}},
			Date: clock.Add(-time.Duration(i) * time.Hour), Preview: "prévia", Body: "Corpo da mensagem " + fmt.Sprint(i),
		}
		if i >= 2 {
			m.Flags = domain.Seen
		}
		b.msgs["INBOX"] = append(b.msgs["INBOX"], m)
		b.byID[m.ID] = m
	}
	return b
}

func (b *fakeBox) Folders() ([]domain.Folder, error) { return b.folders, nil }
func (b *fakeBox) Page(folder string, offset, limit int) ([]domain.Message, int, error) {
	all := b.msgs[folder]
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], len(all), nil
}
func (b *fakeBox) Find(text string, _ int) ([]domain.Message, error) {
	var out []domain.Message
	for _, m := range b.msgs["INBOX"] {
		if strings.Contains(strings.ToLower(m.Subject), strings.ToLower(text)) {
			out = append(out, m)
		}
	}
	return out, nil
}
func (b *fakeBox) Run() (application.Result, error) { b.syncs++; return b.syncResult, b.syncErr }
func (b *fakeBox) Open(id string) (domain.Message, error) {
	return b.byID[id], b.openErr
}
func (b *fakeBox) MarkRead(id string, read bool) error {
	b.marked++
	b.lastRead = read
	return b.markErr
}
func (b *fakeBox) Delete(id string) error { b.deleted++; return b.deleteErr }

func newScreen(b *fakeBox) *Screen {
	s := NewScreen(Config{Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Mailbox: b, Now: func() time.Time { return clock }})
	s.Doc.Layout()
	return s
}

func tapAt(d *widget.Document, p geom.Point) {
	d.HandlePointer(widget.PointerDown, p.X*d.Scale, p.Y*d.Scale)
	d.HandlePointer(widget.PointerUp, p.X*d.Scale, p.Y*d.Scale)
}

func byID(t *testing.T, s *Screen, id string) widget.Widget {
	t.Helper()
	var found widget.Widget
	s.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return true
	})
	if found == nil {
		t.Fatalf("no element %q", id)
	}
	return found
}

func tap(t *testing.T, s *Screen, id string) {
	t.Helper()
	s.Doc.Layout()
	w := byID(t, s, id)
	if got := s.Doc.TargetAt(w.Elem().Bounds.Center()); got == nil {
		t.Fatalf("%s is not reachable", id)
	}
	tapAt(s.Doc, w.Elem().Bounds.Center())
	s.Doc.Layout()
}

func label(t *testing.T, s *Screen, id string) string {
	t.Helper()
	return byID(t, s, id).(*widget.Label).Text
}

func TestEveryTextIsTranslatedInBothLanguages(t *testing.T) {
	tr := NewTranslator("pt-BR")
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks: %v", lang, missing)
		}
	}
}

func TestTheInboxIsShownFirstWithItsUnreadCount(t *testing.T) {
	s := newScreen(newBox(3))
	if s.Folder().ID != "INBOX" {
		t.Errorf("folder = %q", s.Folder().ID)
	}
	if got := label(t, s, "mail-title"); got != "Caixa de entrada  ·  2 não lidas" {
		t.Errorf("title = %q", got)
	}
	byID(t, s, "msg-m00")
	byID(t, s, "msg-m02")
}

func TestTheListPagesThroughTheMessages(t *testing.T) {
	s := newScreen(newBox(10))
	byID(t, s, "msg-m06")
	if s.Page() != 0 || s.pages() != 2 {
		t.Fatalf("page %d of %d", s.Page(), s.pages())
	}
	tap(t, s, "mail-next")
	if s.Page() != 1 {
		t.Errorf("page = %d", s.Page())
	}
	byID(t, s, "msg-m07")
	if !s.Key(true) || s.Page() != 0 {
		t.Errorf("the previous key did not go back: %d", s.Page())
	}
	s.GoTo(99)
	if s.Page() != 1 {
		t.Errorf("clamped page = %d", s.Page())
	}
}

func TestTappingAMessageOpensItAndMarksItRead(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	tap(t, s, "msg-m00")
	if s.Reading() == nil || s.Reading().ID != "m00" {
		t.Fatalf("reading = %v", s.Reading())
	}
	if b.marked != 1 || !b.lastRead {
		t.Errorf("marked = %d read = %v", b.marked, b.lastRead)
	}
	if got := byID(t, s, "mail-text").(*widget.TextPager).Text; got != "Corpo da mensagem 0" {
		t.Errorf("text = %q", got)
	}
	if s.Key(false) {
		t.Error("the page keys must be left to the message while reading")
	}
	tap(t, s, "mail-back")
	if s.Reading() != nil {
		t.Error("still reading")
	}
	byID(t, s, "mail-title")
}

func TestAMessageAlreadyReadIsNotMarkedAgain(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	tap(t, s, "msg-m02")
	if b.marked != 0 {
		t.Errorf("marked = %d", b.marked)
	}
}

func TestMarkUnreadAndDeleteGoBackToTheList(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	tap(t, s, "msg-m02")
	tap(t, s, "mail-unread")
	if b.marked != 1 || b.lastRead || s.Reading() != nil {
		t.Errorf("marked = %d read = %v reading = %v", b.marked, b.lastRead, s.Reading())
	}
	tap(t, s, "msg-m02")
	tap(t, s, "mail-delete")
	if b.deleted != 1 || s.Reading() != nil {
		t.Errorf("deleted = %d reading = %v", b.deleted, s.Reading())
	}
}

func TestADeleteTheServerRefusedKeepsTheMessageOpen(t *testing.T) {
	b := newBox(3)
	b.deleteErr = errors.New("recusado")
	s := newScreen(b)
	tap(t, s, "msg-m02")
	tap(t, s, "mail-delete")
	if s.Reading() == nil {
		t.Error("the message was closed")
	}
	if !strings.Contains(s.cfg.Notice, "recusado") {
		t.Errorf("notice = %q", s.cfg.Notice)
	}
}

func TestOpeningFailsClearly(t *testing.T) {
	b := newBox(3)
	b.openErr = errors.New("sem rede")
	b.byID["m00"] = domain.Message{}
	s := newScreen(b)
	tap(t, s, "msg-m00")
	if s.Reading() != nil || !strings.Contains(s.cfg.Notice, "sem rede") {
		t.Errorf("reading = %v notice = %q", s.Reading(), s.cfg.Notice)
	}
}

func TestSearchListsTheMatchesAndEmptyGoesBack(t *testing.T) {
	s := newScreen(newBox(10))
	s.Search("assunto 3")
	if len(s.items) != 1 || s.items[0].ID != "m03" {
		t.Errorf("items = %+v", s.items)
	}
	s.Search("nada")
	if len(s.items) != 0 {
		t.Errorf("items = %d", len(s.items))
	}
	s.Search("")
	if len(s.items) != rowsPerPage {
		t.Errorf("back to the folder: %d rows", len(s.items))
	}
}

func TestFoldersAreChosenFromTheDropdown(t *testing.T) {
	s := newScreen(newBox(3))
	s.OpenFolder("Sent")
	if s.Folder().ID != "Sent" || label(t, s, "mail-title") != "Enviados" {
		t.Errorf("folder = %q title = %q", s.Folder().ID, label(t, s, "mail-title"))
	}
	s.OpenFolder("Projetos")
	if label(t, s, "mail-title") != "Projetos" {
		t.Errorf("title = %q", label(t, s, "mail-title"))
	}
	s.OpenFolder("nope")
	if s.Folder().ID != "Projetos" {
		t.Error("an unknown folder changed the screen")
	}
}

func TestSyncReportsEachOutcome(t *testing.T) {
	b := newBox(3)
	b.syncResult = application.Result{Folders: 3, Messages: 12}
	s := newScreen(b)
	tap(t, s, "mail-sync")
	if b.syncs != 1 || !strings.Contains(s.cfg.Notice, "12 mensagens em 3 pastas") {
		t.Errorf("syncs = %d notice = %q", b.syncs, s.cfg.Notice)
	}
	b.syncResult = application.Result{Messages: 5, Errors: []error{errors.New("pasta X")}}
	s.Sync()
	if !strings.Contains(s.cfg.Notice, "1 pastas falharam: pasta X") {
		t.Errorf("notice = %q", s.cfg.Notice)
	}
	b.syncErr = domain.ErrUnauthorized
	s.Sync()
	if !strings.Contains(s.cfg.Notice, "recusou o login") {
		t.Errorf("notice = %q", s.cfg.Notice)
	}
	b.syncErr = errors.New("sem rede")
	s.Sync()
	if !strings.Contains(s.cfg.Notice, "sem rede") {
		t.Errorf("notice = %q", s.cfg.Notice)
	}
}

func TestAnEmptyFolderSaysSo(t *testing.T) {
	s := newScreen(newBox(0))
	if got := label(t, s, "mail-title"); got != "Caixa de entrada  ·  2 não lidas" {
		t.Errorf("title = %q", got)
	}
	if len(s.items) != 0 {
		t.Error("rows from nothing")
	}
}

// checkElement verifies one element: it fits the screen, and a touch at its center
// reaches it with a target of at least 56 px.
func checkElement(t *testing.T, s *Screen, where string, w widget.Widget, screenH float64) {
	t.Helper()
	e := w.Elem()
	if !e.Visible() {
		return
	}
	if e.Bounds.Y+e.Bounds.H > screenH+0.5 && e.ID != "" {
		t.Errorf("%s: %s overflows the screen (%v)", where, e.ID, e.Bounds)
	}
	if _, ok := w.(widget.PointerHandler); !ok || e.Disabled || e.Bounds.W <= 0 {
		return
	}
	if got := s.Doc.TargetAt(e.Bounds.Center()); got != w {
		t.Errorf("%s: %s (%T) does not get its own touch", where, e.ID, w)
	}
	if e.Bounds.H < 56 {
		t.Errorf("%s: %s is only %.0f px high", where, e.ID, e.Bounds.H)
	}
}

func TestEveryInteractiveElementReceivesItsOwnTouchAndNothingOverflows(t *testing.T) {
	s := newScreen(newBox(10))
	check := func(where string) {
		t.Helper()
		s.Doc.Layout()
		h := s.Doc.Root.Elem().Bounds.H
		s.Doc.Walk(func(w widget.Widget) bool {
			checkElement(t, s, where, w, h)
			return true
		})
	}
	check("list")
	tap(t, s, "msg-m00")
	check("message")
}

func TestShortDate(t *testing.T) {
	cases := []struct {
		when time.Time
		want string
	}{
		{clock.Add(-3 * time.Hour), "09:00"},
		{clock.AddDate(0, 0, -1), "Ontem"},
		{clock.AddDate(0, 0, -10), "10/05"},
		{clock.AddDate(-1, 0, 0), "20/05/2025"},
		{time.Time{}, ""},
	}
	for _, c := range cases {
		if got := shortDate(c.when, clock, "Ontem"); got != c.want {
			t.Errorf("shortDate(%v) = %q, want %q", c.when, got, c.want)
		}
	}
}

func (b *fakeBox) SetFlagged(id string, on bool) error {
	b.flagged = append(b.flagged, on)
	return b.markErr
}
func (b *fakeBox) New() domain.Draft {
	return domain.Draft{From: domain.Address{Email: "eu@example.org"}}
}
func (b *fakeBox) Reply(id string, all bool, attribution string) (domain.Draft, error) {
	if b.replyErr != nil {
		return domain.Draft{}, b.replyErr
	}
	m := b.byID[id]
	return domain.Draft{
		From: domain.Address{Email: "eu@example.org"}, To: []domain.Address{m.From},
		Subject: "Re: " + m.Subject, Body: "\n\n" + attribution + "\n> " + m.Body,
	}, nil
}
func (b *fakeBox) Forward(id, header string) (domain.Draft, error) {
	return domain.Draft{Subject: "Fwd: " + b.byID[id].Subject, Body: header + "\n" + b.byID[id].Body}, nil
}
func (b *fakeBox) Send(d domain.Draft) error {
	if b.sendErr != nil {
		return b.sendErr
	}
	b.sent = append(b.sent, d)
	return nil
}

func TestReplyFillsTheFormAndSendGoesThrough(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	tap(t, s, "msg-m00")
	tap(t, s, "mail-reply")
	if !s.Composing() {
		t.Fatal("the form did not open")
	}
	if got := byID(t, s, "mail-to").(*widget.Input).Value; got != `"Ana" <ana@example.org>` {
		t.Errorf("to = %q", got)
	}
	if got := byID(t, s, "mail-subject").(*widget.Input).Value; got != "Re: Assunto 0" {
		t.Errorf("subject = %q", got)
	}
	if body := byID(t, s, "mail-body-text").(*widget.TextArea).Value; !strings.Contains(body, "escreveu:") || !strings.Contains(body, "> Corpo") {
		t.Errorf("body = %q", body)
	}
	byID(t, s, "mail-body-text").(*widget.TextArea).Value = "Combinado!"
	tap(t, s, "mail-send")
	if len(b.sent) != 1 || b.sent[0].Body != "Combinado!" || b.sent[0].To[0].Email != "ana@example.org" {
		t.Fatalf("sent = %+v", b.sent)
	}
	if s.Composing() || !strings.Contains(s.cfg.Notice, "enviada") {
		t.Errorf("composing = %v notice = %q", s.Composing(), s.cfg.Notice)
	}
}

func TestNewForwardAndCancel(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	tap(t, s, "mail-new")
	if !s.Composing() || byID(t, s, "mail-to").(*widget.Input).Value != "" {
		t.Error("an empty form was expected")
	}
	tap(t, s, "mail-cancel")
	if s.Composing() || len(b.sent) != 0 {
		t.Error("cancel did not go back")
	}
	tap(t, s, "msg-m00")
	tap(t, s, "mail-forward")
	if got := byID(t, s, "mail-subject").(*widget.Input).Value; got != "Fwd: Assunto 0" {
		t.Errorf("subject = %q", got)
	}
	tap(t, s, "mail-cancel")
	tap(t, s, "msg-m00")
	tap(t, s, "mail-replyall")
	if !s.Composing() {
		t.Error("reply all did not open")
	}
}

func TestASendThatFailsOrABadAddressKeepsTheFormOpen(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	s.NewMessage()
	byID(t, s, "mail-to").(*widget.Input).Value = "isto não é um endereço"
	s.SendDraft()
	if !s.Composing() || len(b.sent) != 0 {
		t.Error("a bad address was sent")
	}
	byID(t, s, "mail-to").(*widget.Input).Value = "bia@example.org"
	b.sendErr = errors.New("servidor fora")
	s.SendDraft()
	if !s.Composing() || !strings.Contains(s.form.note.Text, "servidor fora") {
		t.Errorf("composing = %v note = %q", s.Composing(), s.form.note.Text)
	}
	b.replyErr = errors.New("sem corpo")
	s.CancelCompose()
	tap(t, s, "msg-m00")
	s.Reply(false)
	if s.Composing() || !strings.Contains(s.cfg.Notice, "sem corpo") {
		t.Errorf("composing = %v notice = %q", s.Composing(), s.cfg.Notice)
	}
	var none Screen
	none.Reply(false)
	none.Forward()
	none.SendDraft()
}

func TestTheComposeFormFitsAndEverythingIsReachable(t *testing.T) {
	s := newScreen(newBox(3))
	s.NewMessage()
	s.Doc.Layout()
	h := s.Doc.Root.Elem().Bounds.H
	s.Doc.Walk(func(w widget.Widget) bool {
		checkElement(t, s, "compose", w, h)
		return true
	})
}

func TestTheFlagButtonTogglesAndShowsTheMark(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	tap(t, s, "msg-m02")
	tap(t, s, "mail-flag")
	if len(b.flagged) != 1 || !b.flagged[0] || !s.Reading().Has(domain.Flagged) {
		t.Fatalf("flagged = %v", b.flagged)
	}
	if got := byID(t, s, "mail-flag").(*widget.Button).Text; got != "Tirar sinal" {
		t.Errorf("button = %q", got)
	}
	tap(t, s, "mail-flag")
	if len(b.flagged) != 2 || b.flagged[1] || s.Reading().Has(domain.Flagged) {
		t.Errorf("flagged = %v", b.flagged)
	}
	b.markErr = errors.New("offline")
	tap(t, s, "mail-flag")
	if !strings.Contains(s.cfg.Notice, "offline") {
		t.Errorf("notice = %q", s.cfg.Notice)
	}
	var none Screen
	none.ToggleFlag()
}

func TestBccIsSentAndABadOneStopsTheSend(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	s.NewMessage()
	byID(t, s, "mail-to").(*widget.Input).Value = "bia@example.org"
	byID(t, s, "mail-bcc").(*widget.Input).Value = "cris@example.org, dani@example.org"
	s.SendDraft()
	if len(b.sent) != 1 || len(b.sent[0].Bcc) != 2 {
		t.Fatalf("sent = %+v", b.sent)
	}
	s.NewMessage()
	byID(t, s, "mail-to").(*widget.Input).Value = "bia@example.org"
	byID(t, s, "mail-bcc").(*widget.Input).Value = "lixo"
	s.SendDraft()
	if len(b.sent) != 1 || !s.Composing() {
		t.Error("a bad Bcc was sent")
	}
}

func (b *fakeBox) SaveDraft(d domain.Draft) (domain.Draft, error) {
	if b.draftErr != nil {
		return d, b.draftErr
	}
	if d.ID == "" {
		d.ID = "d" + strconv.Itoa(len(b.drafts)+1)
	}
	b.drafts = append(b.drafts, d)
	return d, nil
}
func (b *fakeBox) DraftList() ([]domain.Draft, error) { return b.drafts, b.draftErr }
func (b *fakeBox) DeleteDraft(string) error           { return nil }

func TestADraftIsSavedListedAndReopened(t *testing.T) {
	b := newBox(3)
	s := newScreen(b)
	tap(t, s, "mail-new")
	byID(t, s, "mail-to").(*widget.Input).Value = "bia@example.org"
	byID(t, s, "mail-subject").(*widget.Input).Value = "ideia"
	tap(t, s, "mail-savedraft")
	if len(b.drafts) != 1 || b.drafts[0].Subject != "ideia" || s.Composing() {
		t.Fatalf("drafts = %+v composing %v", b.drafts, s.Composing())
	}
	if !strings.Contains(s.cfg.Notice, "Rascunho salvo") {
		t.Errorf("notice = %q", s.cfg.Notice)
	}
	tap(t, s, "mail-drafts")
	tap(t, s, "draft-d1")
	if !s.Composing() || byID(t, s, "mail-subject").(*widget.Input).Value != "ideia" {
		t.Error("the draft did not reopen")
	}
	byID(t, s, "mail-to").(*widget.Input).Value = "lixo"
	s.SaveDraft()
	if len(b.drafts) != 1 || !s.Composing() {
		t.Error("a draft with a bad address was kept")
	}
	b.draftErr = errors.New("disco")
	byID(t, s, "mail-to").(*widget.Input).Value = "bia@example.org"
	s.SaveDraft()
	if !s.Composing() || !strings.Contains(s.form.note.Text, "disco") {
		t.Errorf("note = %q", s.form.note.Text)
	}
	s.CancelCompose()
	s.ShowDrafts() // the store fails: a notice, no panic
	if !strings.Contains(s.cfg.Notice, "disco") {
		t.Errorf("notice = %q", s.cfg.Notice)
	}
	var none Screen
	none.SaveDraft()
}

func TestNoDraftsSaysSo(t *testing.T) {
	s := newScreen(newBox(1))
	tap(t, s, "mail-drafts")
	tap(t, s, "drafts-back")
	if s.Composing() {
		t.Error("unexpected form")
	}
}
