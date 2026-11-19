package application

import "go-pocketbook-base/internal/mail/domain"

// Mailbox is what the mail screens use: Sync (Run, Open, MarkRead, Delete) plus
// reading the cache. The screens read only from the store.
type Mailbox struct {
	Sync
	// Me is the account's own address: the sender of what is written, and never a
	// recipient of a reply.
	Me domain.Address
}

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

// Reply starts the answer to a message. attribution introduces the quote and is
// already in the person's language.
func (m Mailbox) Reply(id string, all bool, attribution string) (domain.Draft, error) {
	orig, err := m.Open(id)
	if err != nil {
		return domain.Draft{}, err
	}
	d := domain.Reply(orig, m.Me, all, attribution)
	d.AccountID = m.AccountID
	return d, nil
}

// Forward starts the forward of a message; header introduces the original.
func (m Mailbox) Forward(id, header string) (domain.Draft, error) {
	orig, err := m.Open(id)
	if err != nil {
		return domain.Draft{}, err
	}
	d := domain.Forward(orig, m.Me, header)
	d.AccountID = m.AccountID
	return d, nil
}

// New starts an empty message from the account.
func (m Mailbox) New() domain.Draft { return domain.Draft{AccountID: m.AccountID, From: m.Me} }

// Send checks the draft and sends it through the account's server.
func (m Mailbox) Send(d domain.Draft) error {
	d.AccountID = m.AccountID
	if d.From.Email == "" {
		d.From = m.Me
	}
	if err := d.Validate(); err != nil {
		return err
	}
	return m.Backend.Send(d)
}
