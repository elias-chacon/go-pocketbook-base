package sqlstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"go-pocketbook-base/internal/mail/domain"
)

// SaveDraft implements domain.DraftStore.
func (s *Store) SaveDraft(d domain.Draft) (domain.Draft, error) {
	if d.ID == "" {
		var salt [8]byte
		if _, err := rand.Read(salt[:]); err != nil {
			return d, err
		}
		d.ID = "d" + hex.EncodeToString(salt[:])
	}
	body, err := json.Marshal(d)
	if err != nil {
		return d, err
	}
	_, err = s.db.Exec(`INSERT INTO mail_drafts (id, account_id, saved, body) VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET saved = excluded.saved, body = excluded.body`,
		d.ID, d.AccountID, time.Now().UnixNano(), string(body))
	return d, err
}

// Drafts implements domain.DraftStore.
func (s *Store) Drafts(accountID string) ([]domain.Draft, error) {
	rows, err := s.db.Query("SELECT body FROM mail_drafts WHERE account_id = ? ORDER BY saved DESC", accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Draft
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		var d domain.Draft
		if err := json.Unmarshal([]byte(body), &d); err != nil {
			return nil, fmt.Errorf("mail: a draft is damaged: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// DeleteDraft implements domain.DraftStore.
func (s *Store) DeleteDraft(id string) error {
	res, err := s.db.Exec("DELETE FROM mail_drafts WHERE id = ?", id)
	return affected(res, err)
}
