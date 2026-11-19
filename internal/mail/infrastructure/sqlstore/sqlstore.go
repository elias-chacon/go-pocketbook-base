// Package sqlstore keeps the messages in the shared SQLite database
// (internal/storage): the domain.MessageStore of the reader.
//
// The list queries never read the body (a page of 20 messages must not load 20
// bodies); search is on a column folded to lower case without accents, built when a
// message is saved.
package sqlstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/storage"
)

// Module is the name of this schema in storage.Migrate.
const Module = "mail"

// migrations are the steps of the mail schema.
var migrations = []storage.Migration{
	{Version: 1, SQL: `
CREATE TABLE mail_folders (
	id TEXT NOT NULL, account_id TEXT NOT NULL, name TEXT NOT NULL, role TEXT NOT NULL,
	PRIMARY KEY (account_id, id)
);
CREATE TABLE mail_messages (
	id TEXT PRIMARY KEY, account_id TEXT NOT NULL, folder_id TEXT NOT NULL, remote_id TEXT NOT NULL,
	message_id TEXT NOT NULL DEFAULT '', in_reply_to TEXT NOT NULL DEFAULT '', refs TEXT NOT NULL DEFAULT '[]',
	from_name TEXT NOT NULL DEFAULT '', from_email TEXT NOT NULL DEFAULT '',
	reply_to TEXT NOT NULL DEFAULT '[]', to_list TEXT NOT NULL DEFAULT '[]', cc_list TEXT NOT NULL DEFAULT '[]',
	subject TEXT NOT NULL DEFAULT '', date INTEGER NOT NULL DEFAULT 0, preview TEXT NOT NULL DEFAULT '',
	body TEXT NOT NULL DEFAULT '', flags INTEGER NOT NULL DEFAULT 0, attachments TEXT NOT NULL DEFAULT '[]',
	search TEXT NOT NULL DEFAULT '',
	UNIQUE (account_id, folder_id, remote_id)
);
CREATE INDEX mail_messages_folder ON mail_messages (account_id, folder_id, date DESC);
`},
	{Version: 2, SQL: `
CREATE TABLE mail_drafts (
	id TEXT PRIMARY KEY, account_id TEXT NOT NULL, saved INTEGER NOT NULL, body TEXT NOT NULL
);
CREATE INDEX mail_drafts_account ON mail_drafts (account_id, saved DESC);
`},
}

// Store implements domain.MessageStore.
type Store struct{ db *sql.DB }

// New prepares the mail tables in the database and returns the store.
func New(s *storage.Store) (*Store, error) {
	if err := s.Migrate(Module, migrations); err != nil {
		return nil, err
	}
	for _, table := range []string{"mail_folders", "mail_messages", "mail_drafts"} {
		if err := s.RegisterAccountTable(table); err != nil {
			return nil, err
		}
	}
	return &Store{db: s.DB()}, nil
}

// SaveFolders implements domain.MessageStore.
func (s *Store) SaveFolders(accountID string, folders []domain.Folder) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	keep := make([]interface{}, 0, len(folders)+1)
	keep = append(keep, accountID)
	for _, f := range folders {
		if _, err := tx.Exec(`INSERT INTO mail_folders(id, account_id, name, role) VALUES(?, ?, ?, ?)
			ON CONFLICT(account_id, id) DO UPDATE SET name = excluded.name, role = excluded.role`,
			f.ID, accountID, f.Name, string(f.Role)); err != nil {
			tx.Rollback()
			return err
		}
		keep = append(keep, f.ID)
	}
	// folders (and their messages) that are no longer on the server
	marks := strings.TrimSuffix(strings.Repeat("?,", len(folders)), ",")
	where := "account_id = ?"
	if len(folders) > 0 {
		where += " AND id NOT IN (" + marks + ")"
	}
	if _, err := tx.Exec("DELETE FROM mail_messages WHERE account_id = ? AND folder_id IN (SELECT id FROM mail_folders WHERE "+where+")",
		append([]interface{}{accountID}, keep...)...); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec("DELETE FROM mail_folders WHERE "+where, keep...); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Folders implements domain.MessageStore.
func (s *Store) Folders(accountID string) ([]domain.Folder, error) {
	rows, err := s.db.Query(`
SELECT f.id, f.name, f.role,
       (SELECT COUNT(*) FROM mail_messages m WHERE m.account_id = f.account_id AND m.folder_id = f.id AND (m.flags & ?) = 0),
       (SELECT COUNT(*) FROM mail_messages m WHERE m.account_id = f.account_id AND m.folder_id = f.id)
FROM mail_folders f WHERE f.account_id = ? ORDER BY f.name`, int(domain.Seen), accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Folder
	for rows.Next() {
		f := domain.Folder{AccountID: accountID}
		var role string
		if err := rows.Scan(&f.ID, &f.Name, &role, &f.Unread, &f.Total); err != nil {
			return nil, err
		}
		f.Role = domain.FolderRole(role)
		out = append(out, f)
	}
	return out, rows.Err()
}

// Save implements domain.MessageStore.
func (s *Store) Save(messages ...domain.Message) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, m := range messages {
		if m.ID == "" {
			m.ID = m.AccountID + ":" + m.FolderID + ":" + m.RemoteID
		}
		if err := insert(tx, m); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func insert(tx *sql.Tx, m domain.Message) error {
	refs, _ := json.Marshal(m.References)
	replyTo, _ := json.Marshal(m.ReplyTo)
	to, _ := json.Marshal(m.To)
	cc, _ := json.Marshal(m.Cc)
	attachments, _ := json.Marshal(m.Attachments)
	_, err := tx.Exec(`
INSERT INTO mail_messages(id, account_id, folder_id, remote_id, message_id, in_reply_to, refs, from_name, from_email,
	reply_to, to_list, cc_list, subject, date, preview, body, flags, attachments, search)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(account_id, folder_id, remote_id) DO UPDATE SET
	message_id = excluded.message_id, in_reply_to = excluded.in_reply_to, refs = excluded.refs,
	from_name = excluded.from_name, from_email = excluded.from_email, reply_to = excluded.reply_to,
	to_list = excluded.to_list, cc_list = excluded.cc_list, subject = excluded.subject, date = excluded.date,
	preview = excluded.preview, body = excluded.body, flags = excluded.flags, attachments = excluded.attachments,
	search = excluded.search`,
		m.ID, m.AccountID, m.FolderID, m.RemoteID, m.MessageID, m.InReplyTo, string(refs), m.From.Name, m.From.Email,
		string(replyTo), string(to), string(cc), m.Subject, m.Date.Unix(), m.Preview, m.Body, int(m.Flags),
		string(attachments), fold(m.Subject+" "+m.From.Name+" "+m.From.Email+" "+m.Preview+" "+m.Body))
	return err
}

// listColumns are the columns of a list row: everything but the body.
const listColumns = `id, account_id, folder_id, remote_id, message_id, in_reply_to, refs, from_name, from_email,
	reply_to, to_list, cc_list, subject, date, preview, flags, attachments`

// The two queries of messages: a list row, and a message with its body.
const (
	selectRows     = "SELECT " + listColumns + " FROM mail_messages"
	selectWithBody = "SELECT " + listColumns + ", body FROM mail_messages"
)

// scanner is a sql.Row or sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scan(r scanner, withBody bool) (domain.Message, error) {
	var m domain.Message
	var refs, replyTo, to, cc, attachments string
	var date int64
	var flags int
	dest := []interface{}{&m.ID, &m.AccountID, &m.FolderID, &m.RemoteID, &m.MessageID, &m.InReplyTo, &refs,
		&m.From.Name, &m.From.Email, &replyTo, &to, &cc, &m.Subject, &date, &m.Preview, &flags, &attachments}
	if withBody {
		dest = append(dest, &m.Body)
	}
	if err := r.Scan(dest...); err != nil {
		return domain.Message{}, err
	}
	m.Date, m.Flags = time.Unix(date, 0).UTC(), domain.Flag(flags)
	for target, raw := range map[interface{}]string{&m.References: refs, &m.ReplyTo: replyTo, &m.To: to, &m.Cc: cc, &m.Attachments: attachments} {
		if err := json.Unmarshal([]byte(raw), target); err != nil {
			return domain.Message{}, fmt.Errorf("sqlstore: a damaged row: %w", err)
		}
	}
	return m, nil
}

// Page implements domain.MessageStore.
func (s *Store) Page(q domain.Query) ([]domain.Message, int, error) {
	where, args := "account_id = ? AND folder_id = ?", []interface{}{q.AccountID, q.FolderID}
	if q.UnreadOnly {
		where += " AND (flags & ?) = 0"
		args = append(args, int(domain.Seen))
	}
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM mail_messages WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(selectRows+" WHERE "+where+" ORDER BY date DESC, id LIMIT ? OFFSET ?",
		append(args, q.Page(), offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out, err := collect(rows)
	return out, total, err
}

func collect(rows *sql.Rows) ([]domain.Message, error) {
	var out []domain.Message
	for rows.Next() {
		m, err := scan(rows, false)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Get implements domain.MessageStore.
func (s *Store) Get(id string) (domain.Message, error) {
	m, err := scan(s.db.QueryRow(selectWithBody+" WHERE id = ?", id), true)
	if err == sql.ErrNoRows {
		return domain.Message{}, domain.ErrNotFound
	}
	return m, err
}

// Search implements domain.MessageStore.
func (s *Store) Search(accountID, text string, limit int) ([]domain.Message, error) {
	words := strings.Fields(fold(text))
	if len(words) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > domain.MaxPage {
		limit = domain.DefaultPage
	}
	where, args := "account_id = ?", []interface{}{accountID}
	for _, w := range words {
		where += ` AND search LIKE ? ESCAPE '\'`
		args = append(args, "%"+likeEscape(w)+"%")
	}
	rows, err := s.db.Query(selectRows+" WHERE "+where+" ORDER BY date DESC, id LIMIT ?", append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collect(rows)
}

// SetFlags implements domain.MessageStore.
func (s *Store) SetFlags(id string, add, remove domain.Flag) error {
	res, err := s.db.Exec("UPDATE mail_messages SET flags = (flags | ?) & ~? WHERE id = ?", int(add), int(remove), id)
	return affected(res, err)
}

// Delete implements domain.MessageStore.
func (s *Store) Delete(id string) error {
	res, err := s.db.Exec("DELETE FROM mail_messages WHERE id = ?", id)
	return affected(res, err)
}

func affected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// unaccent folds the accented letters of Portuguese (and the common Latin ones).
var unaccent = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n",
)

// fold is s in lower case without accents: what search compares.
func fold(s string) string { return unaccent.Replace(strings.ToLower(s)) }

// likeEscape makes a word safe inside a LIKE pattern.
func likeEscape(w string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(w)
}
