package application

import "go-pocketbook-base/internal/mail/domain"

// Mailbox is what the mail screens use: Sync (Run, Open, MarkRead, Delete) plus
// reading the cache. The screens read only from the store.
type Mailbox struct{ Sync }

// Folders lists the folders of the account, with their unread and total counts.
func (m Mailbox) Folders() ([]domain.Folder, error) { return m.Store.Folders(m.AccountID) }

// Page lists a page of a folder, newest first, and how many messages it has.
func (m Mailbox) Page(folderID string, offset, limit int) ([]domain.Message, int, error) {
	return m.Store.Page(domain.Query{AccountID: m.AccountID, FolderID: folderID, Offset: offset, Limit: limit})
}

// Find searches the whole account (subject, sender and text).
func (m Mailbox) Find(text string, limit int) ([]domain.Message, error) {
	return m.Store.Search(m.AccountID, text, limit)
}

// Inbox is the folder with the Inbox role, or the first folder when there is none.
func Inbox(folders []domain.Folder) (domain.Folder, bool) {
	for _, f := range folders {
		if f.Role == domain.Inbox {
			return f, true
		}
	}
	if len(folders) > 0 {
		return folders[0], true
	}
	return domain.Folder{}, false
}
