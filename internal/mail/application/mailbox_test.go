package application

import (
	"testing"

	"go-pocketbook-base/internal/mail/domain"
)

func TestMailboxReadsTheCacheOfItsAccount(t *testing.T) {
	st := newMem()
	be := backend()
	mb := Mailbox{Sync{AccountID: "a1", Backend: be, Store: st}}
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
