package domain

import "errors"

// ErrUnauthorized is returned by a Backend when the server refuses the credentials
// (an expired token or a wrong password): the caller renews them and tries again,
// or asks the person to sign in again.
var ErrUnauthorized = errors.New("mail: the server refused the credentials")

// Backend is the port to a mail server (IMAP and SMTP, Microsoft Graph, Exchange).
// A Backend belongs to one account. Messages it returns have no AccountID, FolderID
// is set on the ones that come from a folder, and RemoteID is the server's id.
type Backend interface {
	// Folders lists the folders of the mailbox, with their roles.
	Folders() ([]Folder, error)
	// Recent lists the newest messages of a folder, newest first: headers and a
	// preview, without the body. At most limit.
	Recent(folderID string, limit int) ([]Message, error)
	// Fetch returns one message with its full text.
	Fetch(folderID, remoteID string) (Message, error)
	// Send sends a draft (and keeps a copy in the Sent folder, as the server does).
	Send(d Draft) error
	// SetRead marks a message as read or unread on the server.
	SetRead(folderID, remoteID string, read bool) error
	// Delete moves a message to the trash (or deletes it from the trash).
	Delete(folderID, remoteID string) error
}
