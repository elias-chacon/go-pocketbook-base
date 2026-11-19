package application

import (
	"errors"
	"testing"

	"go-pocketbook-base/internal/mail/domain"
)

func TestMailboxReadsTheCacheOfItsAccount(t *testing.T) {
	st := newMem()
	be := backend()
	mb := Mailbox{Sync: Sync{AccountID: "a1", Backend: be, Store: st}}
	if _, err := mb.Run(); err != nil {
		t.Fatal(err)
	}
	folders, err := mb.Folders()
	if err != nil || len(folders) == 0 {
		t.Fatalf("folders = %v, %v", folders, err)
	}
	in, ok := Inbox(folders)
	if !ok {
		t.Fatal("no inbox")
	}
	page, total, err := mb.Page(in.ID, 0, 10)
	if err != nil || total != len(page) {
		t.Errorf("page = %d of %d, %v", len(page), total, err)
	}
	if _, err := mb.Find("zzz-nothing", 5); err != nil {
		t.Error(err)
	}
}

func TestInboxFallsBackToTheFirstFolder(t *testing.T) {
	if _, ok := Inbox(nil); ok {
		t.Error("an inbox from nothing")
	}
	f, ok := Inbox([]domain.Folder{{ID: "x", Role: domain.Other}, {ID: "y", Role: domain.Sent}})
	if !ok || f.ID != "x" {
		t.Errorf("fallback = %v %v", f, ok)
	}
	f, _ = Inbox([]domain.Folder{{ID: "x", Role: domain.Other}, {ID: "i", Role: domain.Inbox}})
	if f.ID != "i" {
		t.Errorf("inbox = %v", f)
	}
}

func composeBox() (Mailbox, *fakeBackend) {
	st, be := newMem(), backend()
	st.msgs["m1"] = domain.Message{
		ID: "m1", AccountID: "a1", FolderID: "inbox", RemoteID: "1", Subject: "Reunião", Body: "Oi\nTudo bem?", MessageID: "<x@y>",
		From: domain.Address{Name: "Bia", Email: "bia@example.org"},
		To:   []domain.Address{{Email: "eu@example.org"}, {Email: "cris@example.org"}},
	}
	return Mailbox{Sync: Sync{AccountID: "a1", Backend: be, Store: st}, Me: domain.Address{Name: "Eu", Email: "eu@example.org"}}, be
}

func TestReplyAddressesTheSenderAndQuotes(t *testing.T) {
	mb, _ := composeBox()
	d, err := mb.Reply("m1", false, "Bia escreveu:")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.To) != 1 || d.To[0].Email != "bia@example.org" || d.Subject != "Re: Reunião" || d.AccountID != "a1" || d.From.Email != "eu@example.org" {
		t.Errorf("draft = %+v", d)
	}
	all, _ := mb.Reply("m1", true, "x")
	if len(all.To) != 2 {
		t.Errorf("reply all = %+v", all.To)
	}
	if _, err := mb.Reply("nope", false, ""); err == nil {
		t.Error("replied to a message that does not exist")
	}
}

func TestForwardAndNew(t *testing.T) {
	mb, _ := composeBox()
	d, err := mb.Forward("m1", "--- Encaminhada ---")
	if err != nil || d.Subject != "Fwd: Reunião" || len(d.To) != 0 {
		t.Errorf("forward = %+v %v", d, err)
	}
	if _, err := mb.Forward("nope", ""); err == nil {
		t.Error("forwarded a message that does not exist")
	}
	if n := mb.New(); n.From.Email != "eu@example.org" || n.AccountID != "a1" {
		t.Errorf("new = %+v", n)
	}
}

func TestSendChecksTheDraftAndUsesTheBackend(t *testing.T) {
	mb, be := composeBox()
	if err := mb.Send(mb.New()); err == nil {
		t.Error("a draft with no recipient was sent")
	}
	d := mb.New()
	d.From = domain.Address{}
	d.To = []domain.Address{{Email: "bia@example.org"}}
	if err := mb.Send(d); err != nil {
		t.Fatal(err)
	}
	if len(be.sent) != 1 || be.sent[0].From.Email != "eu@example.org" || be.sent[0].AccountID != "a1" {
		t.Errorf("sent = %+v", be.sent)
	}
	be.sendEr = errors.New("recusado")
	if err := mb.Send(d); err == nil || err.Error() != "recusado" {
		t.Errorf("err = %v", err)
	}
}
