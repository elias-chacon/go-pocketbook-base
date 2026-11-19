// Package domain is the heart of the Mail context: messages, folders and drafts,
// and the rules of replying and forwarding, with no knowledge of IMAP, MIME or the
// screen.
//
// Ubiquitous language:
//
//   - Message: a mail someone sent, as the person reads it (text, not MIME).
//   - Folder: where messages are kept (Inbox, Sent...).
//   - Draft: a message being written.
//   - Account: from the Accounts context; a message and a folder belong to one.
package domain

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Address is a person or a mailbox: "Ana Souza <ana@example.org>".
type Address struct {
	Name  string
	Email string
}

// String is the address as written in a header ("Name <email>", or just the email).
func (a Address) String() string {
	if a.Name == "" {
		return a.Email
	}
	return (&mail.Address{Name: a.Name, Address: a.Email}).String()
}

// Display is what the person sees in a list: the name, or the email without a name.
func (a Address) Display() string {
	if strings.TrimSpace(a.Name) != "" {
		return a.Name
	}
	return a.Email
}

// ParseAddress reads "Name <email>" or a bare email.
func ParseAddress(s string) (Address, error) {
	a, err := mail.ParseAddress(strings.TrimSpace(s))
	if err != nil {
		return Address{}, fmt.Errorf("mail: %q is not an address", s)
	}
	return Address{Name: a.Name, Email: a.Address}, nil
}

// Same reports whether two addresses are the same mailbox (case-insensitive).
func (a Address) Same(b Address) bool { return strings.EqualFold(a.Email, b.Email) }

// Flag is something the person or the server marked on a message.
type Flag uint8

// The flags of a message.
const (
	Seen Flag = 1 << iota
	Flagged
	Answered
	DraftFlag
)

// Attachment is a file that came with a message (its content stays on the server
// until the person opens it).
type Attachment struct {
	Name     string
	MIMEType string
	Size     int
}

// Message is a mail as the person reads it.
type Message struct {
	ID        string // local id
	AccountID string
	FolderID  string
	RemoteID  string // the server's id (IMAP UID, Graph id)

	MessageID  string // the Message-ID header
	InReplyTo  string
	References []string

	From        Address
	ReplyTo     []Address
	To, Cc      []Address
	Subject     string
	Date        time.Time
	Preview     string // the first words of the body, for the list
	Body        string // plain text
	Flags       Flag
	Attachments []Attachment
}

// Has reports whether the message has a flag.
func (m Message) Has(f Flag) bool { return m.Flags&f != 0 }

// Unread reports whether the message was not read yet.
func (m Message) Unread() bool { return !m.Has(Seen) }

// FolderRole is what a folder is for, which is how the apps find the Inbox
// whatever the provider calls it.
type FolderRole string

// The roles of a folder.
const (
	Inbox   FolderRole = "inbox"
	Sent    FolderRole = "sent"
	Drafts  FolderRole = "drafts"
	Trash   FolderRole = "trash"
	Junk    FolderRole = "junk"
	Archive FolderRole = "archive"
	Other   FolderRole = "other"
)

// Folder is where messages are kept.
type Folder struct {
	ID        string
	AccountID string
	Name      string
	Role      FolderRole
	Unread    int
	Total     int
}
