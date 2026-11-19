package application

import (
	"errors"

	"go-pocketbook-base/internal/mail/domain"
)

// ErrNoDrafts is returned when saving a draft in a mailbox that keeps none.
var ErrNoDrafts = errors.New("mail: drafts are not kept")

// Mailbox is what the mail screens use: Sync (Run, Open, MarkRead, Delete) plus
// reading the cache. The screens read only from the store.
type Mailbox struct {
	Sync
	// Me is the account's own address: the sender of what is written, and never a
	// recipient of a reply.
	Me domain.Address
	// Drafts keeps the messages being written (nil: drafts are not kept).
	Drafts domain.DraftStore
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

// SaveDraft keeps a message being written, for later.
func (m Mailbox) SaveDraft(d domain.Draft) (domain.Draft, error) {
	if m.Drafts == nil {
		return d, ErrNoDrafts
	}
	d.AccountID = m.AccountID
	if d.From.Email == "" {
		d.From = m.Me
	}
	return m.Drafts.SaveDraft(d)
}

// DraftList returns the drafts, the latest first (none when drafts are not kept).
func (m Mailbox) DraftList() ([]domain.Draft, error) {
	if m.Drafts == nil {
		return nil, nil
	}
	return m.Drafts.Drafts(m.AccountID)
}

// DeleteDraft throws a draft away.
func (m Mailbox) DeleteDraft(id string) error {
	if m.Drafts == nil || id == "" {
		return nil
	}
	return m.Drafts.DeleteDraft(id)
}

// Send checks the draft and sends it through the account's server.
func (m Mailbox) Send(d domain.Draft) error {
	d.AccountID = m.AccountID
	if d.From.Email == "" {
		d.From = m.Me
	}
	if err := d.Validate(); err != nil {
		return err
	}
	if err := m.Backend.Send(d); err != nil {
		return err
	}
	// the message is gone: a draft that cannot be removed must not look like a failed send
	_ = m.DeleteDraft(d.ID)
	return nil
}
