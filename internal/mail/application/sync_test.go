package application

import (
	"errors"
	"testing"

	"go-pocketbook-base/internal/mail/domain"
)

// memStore is a MessageStore in memory.
type memStore struct {
	folders []domain.Folder
	msgs    map[string]domain.Message
	err     error
}

func newMem() *memStore { return &memStore{msgs: map[string]domain.Message{}} }

func (m *memStore) SaveFolders(_ string, f []domain.Folder) error { m.folders = f; return m.err }
func (m *memStore) Folders(string) ([]domain.Folder, error)       { return m.folders, nil }
func (m *memStore) Save(ms ...domain.Message) error {
	if m.err != nil {
		return m.err
	}
	for _, x := range ms {
		if x.ID == "" {
			x.ID = x.AccountID + ":" + x.FolderID + ":" + x.RemoteID
		}
		m.msgs[x.ID] = x
	}
	return nil
}
func (m *memStore) Page(domain.Query) ([]domain.Message, int, error) { return nil, 0, nil }
func (m *memStore) Get(id string) (domain.Message, error) {
	if x, ok := m.msgs[id]; ok {
		return x, nil
	}
	return domain.Message{}, domain.ErrNotFound
}
func (m *memStore) Search(string, string, int) ([]domain.Message, error) { return nil, nil }
func (m *memStore) SetFlags(id string, add, remove domain.Flag) error {
	x, ok := m.msgs[id]
	if !ok {
		return domain.ErrNotFound
	}
	x.Flags = (x.Flags | add) &^ remove
	m.msgs[id] = x
	return nil
}
func (m *memStore) Delete(id string) error {
	if _, ok := m.msgs[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.msgs, id)
	return nil
}

// fakeBackend is a Backend with canned answers that records what it was asked.
type fakeBackend struct {
	folders   []domain.Folder
	recent    map[string][]domain.Message
	fetched   map[string]domain.Message
	foldersEr error
	recentEr  map[string]error
	fetchEr   error
	readEr    error
	deleteEr  error
	limits    []int
	reads     []string
	deleted   []string
}

func (f *fakeBackend) Folders() ([]domain.Folder, error) { return f.folders, f.foldersEr }
func (f *fakeBackend) Recent(folder string, limit int) ([]domain.Message, error) {
	f.limits = append(f.limits, limit)
	if err := f.recentEr[folder]; err != nil {
		return nil, err
	}
	return append([]domain.Message(nil), f.recent[folder]...), nil
}
func (f *fakeBackend) Fetch(folder, remote string) (domain.Message, error) {
	return f.fetched[folder+"/"+remote], f.fetchEr
}
func (f *fakeBackend) Send(domain.Draft) error { return nil }
func (f *fakeBackend) SetRead(folder, remote string, read bool) error {
	f.reads = append(f.reads, folder+"/"+remote)
	return f.readEr
}
func (f *fakeBackend) Delete(folder, remote string) error {
	f.deleted = append(f.deleted, folder+"/"+remote)
	return f.deleteEr
}

func backend() *fakeBackend {
	return &fakeBackend{
		folders: []domain.Folder{{ID: "inbox", Name: "Inbox", Role: domain.Inbox}, {ID: "sent", Name: "Sent", Role: domain.Sent}},
		recent: map[string][]domain.Message{
			"inbox": {{RemoteID: "1", Subject: "Um", Preview: "prévia um"}, {RemoteID: "2", Subject: "Dois", Preview: "prévia dois"}},
			"sent":  {{RemoteID: "9", Subject: "Enviada"}},
		},
		fetched: map[string]domain.Message{"inbox/1": {Subject: "Um", Body: "o corpo inteiro", Attachments: []domain.Attachment{{Name: "a.pdf", Size: 3}}}},
	}
}

func TestSyncBringsTheFoldersAndTheNewestMessagesOfEach(t *testing.T) {
	be, st := backend(), newMem()
	res, err := Sync{AccountID: "ana", Backend: be, Store: st}.Run()
	if err != nil || res.Folders != 2 || res.Messages != 3 || len(res.Errors) != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	if len(st.folders) != 2 || st.folders[0].AccountID != "ana" {
		t.Errorf("the folders carry the account: %+v", st.folders)
	}
	m, err := st.Get("ana:inbox:2")
	if err != nil || m.Subject != "Dois" || m.AccountID != "ana" || m.FolderID != "inbox" {
		t.Errorf("%+v %v", m, err)
	}
	if be.limits[0] != DefaultPerFolder {
		t.Errorf("the default number of messages: %v", be.limits)
	}
	(Sync{AccountID: "ana", Backend: be, Store: st, PerFolder: 7}).Run()
	if be.limits[len(be.limits)-1] != 7 {
		t.Errorf("%v", be.limits)
	}
}

func TestAFailingFolderIsReportedAndTheOthersStillSync(t *testing.T) {
	be, st := backend(), newMem()
	be.recentEr = map[string]error{"inbox": errors.New("timeout")}
	res, err := Sync{AccountID: "ana", Backend: be, Store: st}.Run()
	if err != nil || res.Messages != 1 || len(res.Errors) != 1 || res.Errors[0].Error() != `folder "Inbox": timeout` {
		t.Errorf("%+v %v", res, err)
	}
}

func TestSyncStopsWhenTheFoldersCannotBeListedOrKept(t *testing.T) {
	be := backend()
	be.foldersEr = domain.ErrUnauthorized
	if _, err := (Sync{Backend: be, Store: newMem()}).Run(); err != domain.ErrUnauthorized {
		t.Errorf("%v", err)
	}
	st := newMem()
	st.err = errors.New("disk full")
	if _, err := (Sync{Backend: backend(), Store: st}).Run(); err == nil {
		t.Error("a store that cannot keep the folders")
	}
}

func TestAMessageAlreadyReadInFullKeepsItsBodyWhenTheListComesAgain(t *testing.T) {
	be, st := backend(), newMem()
	s := Sync{AccountID: "ana", Backend: be, Store: st}
	s.Run()
	if _, err := s.Open("ana:inbox:1"); err != nil {
		t.Fatal(err)
	}
	s.Run() // the server list has only the preview
	m, _ := st.Get("ana:inbox:1")
	if m.Body != "o corpo inteiro" {
		t.Errorf("the body survives a new synchronization: %q", m.Body)
	}
}

func TestOpenFetchesTheBodyOnceAndThenOpensOffline(t *testing.T) {
	be, st := backend(), newMem()
	s := Sync{AccountID: "ana", Backend: be, Store: st}
	s.Run()
	st.SetFlags("ana:inbox:1", domain.Seen|domain.Flagged, 0)

	m, err := s.Open("ana:inbox:1")
	if err != nil || m.Body != "o corpo inteiro" || len(m.Attachments) != 1 {
		t.Fatalf("%+v %v", m, err)
	}
	if m.ID != "ana:inbox:1" || m.RemoteID != "1" || m.FolderID != "inbox" || !m.Has(domain.Flagged) || m.Preview != "prévia um" {
		t.Errorf("the local identity, flags and preview are kept: %+v", m)
	}
	be.fetchEr = errors.New("offline")
	again, err := s.Open("ana:inbox:1")
	if err != nil || again.Body != "o corpo inteiro" {
		t.Errorf("opens offline the second time: %v", err)
	}
}

func TestOpeningWithoutTheServerStillShowsTheRowAndTheErrorIsReported(t *testing.T) {
	be, st := backend(), newMem()
	s := Sync{AccountID: "ana", Backend: be, Store: st}
	s.Run()
	be.fetchEr = domain.ErrUnauthorized
	m, err := s.Open("ana:inbox:2")
	if err != domain.ErrUnauthorized || m.Subject != "Dois" {
		t.Errorf("%+v %v", m, err)
	}
	if _, err := s.Open("ghost"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
	be.fetchEr = nil
	be.fetched["inbox/2"] = domain.Message{Subject: "Dois", Body: "corpo"}
	st.err = errors.New("disk full")
	if _, err := s.Open("ana:inbox:2"); err == nil {
		t.Error("a store that cannot keep what was fetched")
	}
}

func TestMarkingReadChangesTheStoreFirstAndThenTheServer(t *testing.T) {
	be, st := backend(), newMem()
	s := Sync{AccountID: "ana", Backend: be, Store: st}
	s.Run()
	if err := s.MarkRead("ana:inbox:1", true); err != nil {
		t.Fatal(err)
	}
	if m, _ := st.Get("ana:inbox:1"); !m.Has(domain.Seen) || len(be.reads) != 1 || be.reads[0] != "inbox/1" {
		t.Errorf("%+v %v", m, be.reads)
	}
	be.readEr = errors.New("offline")
	if err := s.MarkRead("ana:inbox:1", false); err == nil {
		t.Error("an unreachable server is reported")
	}
	if m, _ := st.Get("ana:inbox:1"); m.Has(domain.Seen) {
		t.Error("but the screen already shows it unread")
	}
	if err := s.MarkRead("ghost", true); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}

func TestDeleteRemovesFromTheServerBeforeTheStore(t *testing.T) {
	be, st := backend(), newMem()
	s := Sync{AccountID: "ana", Backend: be, Store: st}
	s.Run()
	be.deleteEr = errors.New("refused")
	if err := s.Delete("ana:inbox:1"); err == nil {
		t.Fatal("refused")
	}
	if _, err := st.Get("ana:inbox:1"); err != nil {
		t.Error("the server refused: the message stays on the screen")
	}
	be.deleteEr = nil
	if err := s.Delete("ana:inbox:1"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get("ana:inbox:1"); err != domain.ErrNotFound || be.deleted[len(be.deleted)-1] != "inbox/1" {
		t.Errorf("%v %v", err, be.deleted)
	}
	if err := s.Delete("ghost"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}
