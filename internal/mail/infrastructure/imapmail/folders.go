package imapmail

import (
	"strings"

	"github.com/emersion/go-imap"

	"go-pocketbook-base/internal/mail/domain"
)

// folderOf turns a mailbox into a Folder; a mailbox that cannot hold messages
// (\Noselect, a bare parent) is skipped.
func folderOf(info *imap.MailboxInfo) (domain.Folder, bool) {
	for _, a := range info.Attributes {
		if strings.EqualFold(a, imap.NoSelectAttr) {
			return domain.Folder{}, false
		}
	}
	return domain.Folder{ID: info.Name, Name: displayName(info), Role: roleOf(info)}, true
}

// displayName is the last part of a path such as "INBOX/Work".
func displayName(info *imap.MailboxInfo) string {
	name := info.Name
	if d := info.Delimiter; d != "" {
		if i := strings.LastIndex(name, d); i >= 0 && i+len(d) < len(name) {
			name = name[i+len(d):]
		}
	}
	if strings.EqualFold(name, "INBOX") {
		return "INBOX"
	}
	return name
}

// special-use attributes (RFC 6154) say what a folder is for in any language.
var attributeRoles = map[string]domain.FolderRole{
	`\sent`:    domain.Sent,
	`\drafts`:  domain.Drafts,
	`\trash`:   domain.Trash,
	`\junk`:    domain.Junk,
	`\archive`: domain.Archive,
	`\all`:     domain.Archive,
}

// nameRoles is the fallback for servers without special-use: the usual names.
var nameRoles = map[string]domain.FolderRole{
	"inbox": domain.Inbox,
	"sent":  domain.Sent, "sent items": domain.Sent, "sent mail": domain.Sent, "enviados": domain.Sent, "itens enviados": domain.Sent,
	"drafts": domain.Drafts, "rascunhos": domain.Drafts,
	"trash": domain.Trash, "deleted items": domain.Trash, "deleted": domain.Trash, "lixeira": domain.Trash, "itens excluídos": domain.Trash,
	"junk": domain.Junk, "spam": domain.Junk, "junk email": domain.Junk,
	"archive": domain.Archive, "arquivo": domain.Archive, "arquivos": domain.Archive,
}

func roleOf(info *imap.MailboxInfo) domain.FolderRole {
	if strings.EqualFold(info.Name, "INBOX") {
		return domain.Inbox
	}
	for _, a := range info.Attributes {
		if r, ok := attributeRoles[strings.ToLower(a)]; ok {
			return r
		}
	}
	if r, ok := nameRoles[strings.ToLower(displayName(info))]; ok {
		return r
	}
	return domain.Other
}

// rank orders the folders the way people look for them: Inbox first.
func rank(r domain.FolderRole) int {
	switch r {
	case domain.Inbox:
		return 0
	case domain.Drafts:
		return 1
	case domain.Sent:
		return 2
	case domain.Archive:
		return 3
	case domain.Junk:
		return 4
	case domain.Trash:
		return 5
	}
	return 6
}
