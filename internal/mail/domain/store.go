package domain

import "errors"

// ErrNotFound is returned for a message or folder that is not in the store.
var ErrNotFound = errors.New("mail: not found")

// Query asks for a page of the messages of a folder, newest first.
type Query struct {
	AccountID  string
	FolderID   string
	UnreadOnly bool
	Offset     int
	Limit      int // at most MaxPage; 0 means DefaultPage
}

// The sizes of a page of messages.
const (
	DefaultPage = 20
	MaxPage     = 200
)

// Page is the effective number of messages the query asks for.
func (q Query) Page() int {
	switch {
	case q.Limit <= 0:
		return DefaultPage
	case q.Limit > MaxPage:
		return MaxPage
	}
	return q.Limit
}

// MessageStore is the port to the messages kept on the reader. The screens read only
// from it (offline-first); the synchronization writes to it.
type MessageStore interface {
	// SaveFolders replaces the folders of an account with the ones the server has;
	// the messages of a folder that no longer exists go with it.
	SaveFolders(accountID string, folders []Folder) error
	// Folders lists the folders of an account with their unread and total counts.
	Folders(accountID string) ([]Folder, error)

	// Save keeps messages, replacing the ones with the same account, folder and
	// remote id. A message with no ID gets one.
	Save(messages ...Message) error
	// Page lists a page of a folder, newest first, and how many messages the folder
	// has in all. The messages have no Body: the list shows the Preview.
	Page(q Query) ([]Message, int, error)
	// Get returns one message with its Body.
	Get(id string) (Message, error)
	// Search finds messages whose subject, sender or text contain every word, newest
	// first (accents and case do not matter), without their Body.
	Search(accountID, text string, limit int) ([]Message, error)

	// SetFlags adds and removes flags.
	SetFlags(id string, add, remove Flag) error
	// Delete removes a message.
	Delete(id string) error
}
