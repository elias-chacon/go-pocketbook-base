// Package application holds the use cases of the Mail context: synchronizing the
// local cache with the server, and opening a message. The screens call these and
// read only from the store.
package application

import (
	"fmt"

	"go-pocketbook-base/internal/mail/domain"
)

// DefaultPerFolder is how many of the newest messages of each folder are kept.
const DefaultPerFolder = 50

// Sync is the use case "bring the cache up to date": the folders and the newest
// messages of each, from the server into the store.
type Sync struct {
	AccountID string
	Backend   domain.Backend
	Store     domain.MessageStore
	// PerFolder is how many messages of each folder to bring (DefaultPerFolder when 0).
	// The reader keeps the newest ones: the rest stays on the server.
	PerFolder int
}

// Result is what a synchronization did.
type Result struct {
	Folders  int
	Messages int
	// Errors are the folders that could not be synchronized: one failing folder does
	// not stop the others.
	Errors []error
}

// Run synchronizes. A failure to list the folders ends it (there is nothing to
// walk); a failure in one folder is reported in the Result.
func (s Sync) Run() (Result, error) {
	folders, err := s.Backend.Folders()
	if err != nil {
		return Result{}, err
	}
	for i := range folders {
		folders[i].AccountID = s.AccountID
	}
	if err := s.Store.SaveFolders(s.AccountID, folders); err != nil {
		return Result{}, err
	}
	limit := s.PerFolder
	if limit <= 0 {
		limit = DefaultPerFolder
	}
	res := Result{Folders: len(folders)}
	for _, f := range folders {
		n, err := s.folder(f, limit)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("folder %q: %w", f.Name, err))
			continue
		}
		res.Messages += n
	}
	return res, nil
}

func (s Sync) folder(f domain.Folder, limit int) (int, error) {
	messages, err := s.Backend.Recent(f.ID, limit)
	if err != nil {
		return 0, err
	}
	for i := range messages {
		messages[i].AccountID, messages[i].FolderID = s.AccountID, f.ID
		// a message already read in full keeps its body: the server list has only a preview
		if old, err := s.Store.Get(messages[i].AccountID + ":" + f.ID + ":" + messages[i].RemoteID); err == nil && messages[i].Body == "" {
			messages[i].Body = old.Body
		}
	}
	return len(messages), s.Store.Save(messages...)
}

// Open returns a message with its body: from the store when it was read before,
// otherwise from the server (and kept, so it opens offline next time).
func (s Sync) Open(id string) (domain.Message, error) {
	m, err := s.Store.Get(id)
	if err != nil {
		return domain.Message{}, err
	}
	if m.Body != "" {
		return m, nil
	}
	full, err := s.Backend.Fetch(m.FolderID, m.RemoteID)
	if err != nil {
		return m, err // the list row is still there to show, with its preview
	}
	// the server's version has the full text and the real attachments; the local flags win
	full.ID, full.AccountID, full.FolderID, full.RemoteID, full.Flags = m.ID, m.AccountID, m.FolderID, m.RemoteID, m.Flags
	if full.Preview == "" {
		full.Preview = m.Preview
	}
	if err := s.Store.Save(full); err != nil {
		return full, err
	}
	return full, nil
}

// MarkRead marks a message read or unread in the store and on the server. The
// store changes first, so the screen answers at once; a server that cannot be
// reached is reported, and the next synchronization sets the flag from the server.
func (s Sync) MarkRead(id string, read bool) error {
	m, err := s.Store.Get(id)
	if err != nil {
		return err
	}
	add, remove := domain.Seen, domain.Flag(0)
	if !read {
		add, remove = 0, domain.Seen
	}
	if err := s.Store.SetFlags(id, add, remove); err != nil {
		return err
	}
	return s.Backend.SetRead(m.FolderID, m.RemoteID, read)
}

// Delete removes a message from the server and then from the store: if the
// server refuses, the message is still on the screen.
func (s Sync) Delete(id string) error {
	m, err := s.Store.Get(id)
	if err != nil {
		return err
	}
	if err := s.Backend.Delete(m.FolderID, m.RemoteID); err != nil {
		return err
	}
	return s.Store.Delete(id)
}
