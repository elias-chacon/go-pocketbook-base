package sqlstore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/storage"
)

func newStore(t *testing.T) (*Store, *storage.Store) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	return s, db
}

func at(day int) time.Time { return time.Date(2026, 1, day, 10, 0, 0, 0, time.UTC) }

func message(remote string, day int, subject string) domain.Message {
	return domain.Message{
		AccountID: "ana", FolderID: "inbox", RemoteID: remote, Subject: subject, Date: at(day),
		From: domain.Address{Name: "Beatriz Gonçalves", Email: "bia@example.org"}, Body: "corpo de " + subject, Preview: "prévia de " + subject,
	}
}

func subjects(ms []domain.Message) string {
	var s []string
	for _, m := range ms {
		s = append(s, m.Subject)
	}
	return strings.Join(s, "|")
}

func TestAMessageRoundTripsWithEveryField(t *testing.T) {
	s, _ := newStore(t)
	in := domain.Message{
		AccountID: "ana", FolderID: "inbox", RemoteID: "42", MessageID: "<m1@x>", InReplyTo: "<m0@x>", References: []string{"<m00@x>", "<m0@x>"},
		From: domain.Address{Name: "Beatriz", Email: "bia@example.org"}, ReplyTo: []domain.Address{{Email: "lista@example.org"}},
		To: []domain.Address{{Name: "Eu", Email: "eu@example.org"}, {Email: "c@example.org"}}, Cc: []domain.Address{{Email: "d@example.org"}},
		Subject: "Reunião", Date: at(5), Preview: "Podemos?", Body: "Podemos nos falar às 10h?", Flags: domain.Seen | domain.Flagged,
		Attachments: []domain.Attachment{{Name: "a.pdf", MIMEType: "application/pdf", Size: 12}},
	}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("ana:inbox:42")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "ana:inbox:42" || got.Body != in.Body || got.Subject != in.Subject || !got.Date.Equal(in.Date) || got.Flags != in.Flags {
		t.Errorf("%+v", got)
	}
	if got.From != in.From || len(got.To) != 2 || got.To[1].Email != "c@example.org" || got.Cc[0].Email != "d@example.org" || got.ReplyTo[0].Email != "lista@example.org" {
		t.Errorf("people: %+v", got)
	}
	if got.InReplyTo != "<m0@x>" || len(got.References) != 2 || len(got.Attachments) != 1 || got.Attachments[0].Size != 12 {
		t.Errorf("thread and attachments: %+v", got)
	}
	if _, err := s.Get("nope"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}

func TestSavingAgainReplacesTheMessageInsteadOfDuplicatingIt(t *testing.T) {
	s, _ := newStore(t)
	s.Save(message("1", 1, "Antes"))
	changed := message("1", 1, "Depois")
	changed.Flags = domain.Seen
	if err := s.Save(changed); err != nil {
		t.Fatal(err)
	}
	page, total, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox"})
	if total != 1 || len(page) != 1 || page[0].Subject != "Depois" || !page[0].Has(domain.Seen) {
		t.Errorf("%d %v", total, subjects(page))
	}
}

func TestAPageIsNewestFirstWithoutBodiesAndTheTotal(t *testing.T) {
	s, _ := newStore(t)
	for i := 1; i <= 7; i++ {
		s.Save(message(fmt.Sprint(i), i, fmt.Sprintf("Assunto %d", i)))
	}
	s.Save(domain.Message{AccountID: "ana", FolderID: "sent", RemoteID: "9", Subject: "Enviada", Date: at(20)})
	s.Save(domain.Message{AccountID: "bia", FolderID: "inbox", RemoteID: "9", Subject: "De outra conta", Date: at(21)})

	page, total, err := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox", Limit: 3})
	if err != nil || total != 7 || subjects(page) != "Assunto 7|Assunto 6|Assunto 5" {
		t.Fatalf("%d %q %v", total, subjects(page), err)
	}
	if page[0].Body != "" || page[0].Preview != "prévia de Assunto 7" {
		t.Errorf("a list row has the preview and no body: %+v", page[0])
	}
	second, _, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox", Limit: 3, Offset: 3})
	if subjects(second) != "Assunto 4|Assunto 3|Assunto 2" {
		t.Errorf("%s", subjects(second))
	}
	last, _, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox", Limit: 3, Offset: 6})
	if subjects(last) != "Assunto 1" {
		t.Errorf("%s", subjects(last))
	}
	if all, _, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox", Offset: -5}); len(all) != 7 {
		t.Errorf("a negative offset is the start, the default page holds 20: %d", len(all))
	}
}

func TestQueryPageSizesAreBounded(t *testing.T) {
	cases := map[int]int{0: domain.DefaultPage, -3: domain.DefaultPage, 5: 5, 5000: domain.MaxPage}
	for in, want := range cases {
		if got := (domain.Query{Limit: in}).Page(); got != want {
			t.Errorf("%d -> %d", in, got)
		}
	}
}

func TestUnreadOnlyAndFlags(t *testing.T) {
	s, _ := newStore(t)
	for i := 1; i <= 4; i++ {
		s.Save(message(fmt.Sprint(i), i, fmt.Sprintf("M%d", i)))
	}
	if err := s.SetFlags("ana:inbox:2", domain.Seen|domain.Flagged, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetFlags("ana:inbox:3", domain.Seen, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetFlags("ana:inbox:3", 0, domain.Seen); err != nil { // marked unread again
		t.Fatal(err)
	}
	unread, total, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox", UnreadOnly: true})
	if total != 3 || subjects(unread) != "M4|M3|M1" {
		t.Errorf("%d %s", total, subjects(unread))
	}
	m, _ := s.Get("ana:inbox:2")
	if !m.Has(domain.Seen) || !m.Has(domain.Flagged) || m.Has(domain.Answered) {
		t.Errorf("%v", m.Flags)
	}
	if err := s.SetFlags("ghost", domain.Seen, 0); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}

func TestDeleteRemovesOneMessage(t *testing.T) {
	s, _ := newStore(t)
	s.Save(message("1", 1, "A"), message("2", 2, "B"))
	if err := s.Delete("ana:inbox:1"); err != nil {
		t.Fatal(err)
	}
	if page, total, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox"}); total != 1 || subjects(page) != "B" {
		t.Errorf("%d %s", total, subjects(page))
	}
	if err := s.Delete("ana:inbox:1"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}

func TestSearchIgnoresCaseAndAccentsAndNeedsEveryWord(t *testing.T) {
	s, _ := newStore(t)
	a := message("1", 1, "Reunião de planejamento")
	b := message("2", 2, "Fatura de janeiro")
	b.From = domain.Address{Name: "Banco", Email: "banco@example.org"}
	b.Body = "O valor de 100% do total_geral"
	s.Save(a, b)

	cases := map[string]string{
		"reuniao":        "Reunião de planejamento", // accents
		"REUNIÃO PLANEJ": "Reunião de planejamento", // case, two words, a prefix
		"goncalves":      "Reunião de planejamento", // the sender of the first message (the second one is from "Banco")
		"banco fatura":   "Fatura de janeiro",
		"100%":           "Fatura de janeiro", // a % in the text is not a wildcard
		"total_geral":    "Fatura de janeiro",
		"reuniao fatura": "",
		"nada disso":     "",
		"   ":            "",
	}
	for q, want := range cases {
		got, err := s.Search("ana", q, 10)
		if err != nil || subjects(got) != want {
			t.Errorf("%q: %q %v, want %q", q, subjects(got), err, want)
		}
	}
	if got, _ := s.Search("bia", "reuniao", 10); len(got) != 0 {
		t.Error("another account's messages are not found")
	}
	if got, _ := s.Search("ana", "goncalves", 1); len(got) != 1 {
		t.Error("the limit applies")
	}
	if _, err := s.Search("ana", "_", 0); err != nil {
		t.Errorf("a wildcard character alone is just a character: %v", err)
	}
}

func TestFoldersCountUnreadAndTotalAndFollowTheServer(t *testing.T) {
	s, _ := newStore(t)
	folders := []domain.Folder{{ID: "inbox", Name: "Caixa de entrada", Role: domain.Inbox}, {ID: "sent", Name: "Enviados", Role: domain.Sent}, {ID: "old", Name: "Antigos", Role: domain.Archive}}
	if err := s.SaveFolders("ana", folders); err != nil {
		t.Fatal(err)
	}
	s.Save(message("1", 1, "A"), message("2", 2, "B"), message("3", 3, "C"))
	s.Save(domain.Message{AccountID: "ana", FolderID: "old", RemoteID: "1", Subject: "Velha", Date: at(1)})
	s.SetFlags("ana:inbox:1", domain.Seen, 0)

	got, err := s.Folders("ana")
	if err != nil || len(got) != 3 {
		t.Fatalf("%v %v", got, err)
	}
	by := map[string]domain.Folder{}
	for _, f := range got {
		by[f.ID] = f
	}
	if by["inbox"].Total != 3 || by["inbox"].Unread != 2 || by["inbox"].Role != domain.Inbox || by["sent"].Total != 0 || by["old"].Total != 1 {
		t.Errorf("%+v", by)
	}
	if got[0].Name != "Antigos" || got[1].Name != "Caixa de entrada" {
		t.Errorf("by name: %+v", got)
	}

	// the server dropped "old" and renamed "sent": its messages go, the others stay
	folders = []domain.Folder{{ID: "inbox", Name: "Inbox", Role: domain.Inbox}, {ID: "sent", Name: "Sent items", Role: domain.Sent}}
	if err := s.SaveFolders("ana", folders); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Folders("ana")
	if len(got) != 2 || got[0].Name != "Inbox" || got[0].Total != 3 || got[1].Name != "Sent items" {
		t.Errorf("%+v", got)
	}
	if _, total, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "old"}); total != 0 {
		t.Error("a folder that is gone takes its messages")
	}

	if err := s.SaveFolders("ana", nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Folders("ana"); len(got) != 0 {
		t.Error("an account with no folders")
	}
	if _, total, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox"}); total != 0 {
		t.Error("and no messages")
	}
}

func TestRemovingAnAccountErasesItsMessagesAndFolders(t *testing.T) {
	s, db := newStore(t)
	s.SaveFolders("ana", []domain.Folder{{ID: "inbox", Name: "Inbox", Role: domain.Inbox}})
	s.SaveFolders("bia", []domain.Folder{{ID: "inbox", Name: "Inbox", Role: domain.Inbox}})
	s.Save(message("1", 1, "da Ana"))
	other := message("1", 1, "da Bia")
	other.AccountID = "bia"
	s.Save(other)

	if err := db.Erase("ana"); err != nil {
		t.Fatal(err)
	}
	if _, total, _ := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox"}); total != 0 {
		t.Error("ana's messages are gone")
	}
	if folders, _ := s.Folders("ana"); len(folders) != 0 {
		t.Error("and her folders")
	}
	if _, total, _ := s.Page(domain.Query{AccountID: "bia", FolderID: "inbox"}); total != 1 {
		t.Error("bia's are not touched")
	}
}

func TestADamagedRowIsReportedNotHidden(t *testing.T) {
	s, db := newStore(t)
	s.Save(message("1", 1, "A"))
	db.DB().Exec(`UPDATE mail_messages SET to_list = 'not json'`)
	if _, err := s.Get("ana:inbox:1"); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Errorf("%v", err)
	}
	if _, _, err := s.Page(domain.Query{AccountID: "ana", FolderID: "inbox"}); err == nil {
		t.Error("a list with a damaged row is an error")
	}
}

func TestTheStoreFailsCleanlyWhenTheDatabaseIsClosed(t *testing.T) {
	s, db := newStore(t)
	db.Close()
	if err := s.Save(message("1", 1, "A")); err == nil {
		t.Error("Save")
	}
	if err := s.SaveFolders("ana", nil); err == nil {
		t.Error("SaveFolders")
	}
	if _, err := s.Folders("ana"); err == nil {
		t.Error("Folders")
	}
	if _, _, err := s.Page(domain.Query{}); err == nil {
		t.Error("Page")
	}
	if _, err := s.Search("ana", "x", 1); err == nil {
		t.Error("Search")
	}
	if err := s.SetFlags("x", 1, 0); err == nil || err == domain.ErrNotFound {
		t.Errorf("SetFlags: %v", err)
	}
}

func TestDraftsRoundTripNewestFirstAndAreErasedWithTheAccount(t *testing.T) {
	s, db := newStore(t)
	d1, err := s.SaveDraft(domain.Draft{AccountID: "ana", Subject: "um", To: []domain.Address{{Email: "a@b.org"}}, Attachments: []domain.DraftAttachment{{Name: "x.txt", Data: []byte("oi")}}})
	if err != nil || d1.ID == "" {
		t.Fatalf("d1 = %+v %v", d1, err)
	}
	d2, _ := s.SaveDraft(domain.Draft{AccountID: "ana", Subject: "dois"})
	d1.Subject = "um editado"
	if _, err := s.SaveDraft(d1); err != nil { // same id: replaces, and is the newest now
		t.Fatal(err)
	}
	list, err := s.Drafts("ana")
	if err != nil || len(list) != 2 || list[0].ID != d1.ID || list[0].Subject != "um editado" || string(list[0].Attachments[0].Data) != "oi" || list[1].ID != d2.ID {
		t.Fatalf("list = %+v %v", list, err)
	}
	if other, _ := s.Drafts("bia"); len(other) != 0 {
		t.Error("drafts of another account")
	}
	if err := s.DeleteDraft(d2.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteDraft(d2.ID); err != domain.ErrNotFound {
		t.Errorf("err = %v", err)
	}
	if err := db.EraseAccount("ana"); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.Drafts("ana"); len(list) != 0 {
		t.Error("the drafts survived the account")
	}
}
