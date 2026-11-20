// Package sqlstore keeps the Bible on the reader in the shared SQLite database
// (internal/storage): the domain.Library. A version is stored one verse per row,
// keyed by (version, book, chapter, verse), so a chapter is one range read and the
// reader never loads a whole version into memory. What the person has read is kept
// by place only (book, chapter, verse), without the version: reading a verse in any
// translation counts for all of them.
package sqlstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/storage"
)

// Module is the name of this schema in storage.Migrate.
const Module = "bible"

var migrations = []storage.Migration{
	{Version: 1, SQL: `
CREATE TABLE bible_books (
	id INTEGER PRIMARY KEY, slug TEXT NOT NULL, chapters INTEGER NOT NULL, new_testament INTEGER NOT NULL DEFAULT 0,
	names TEXT NOT NULL DEFAULT '{}'
);
CREATE TABLE bible_versions (
	slug TEXT PRIMARY KEY, name TEXT NOT NULL, short_name TEXT NOT NULL DEFAULT '', language TEXT NOT NULL,
	books INTEGER NOT NULL DEFAULT 0, chapters INTEGER NOT NULL DEFAULT 0,
	downloaded INTEGER NOT NULL DEFAULT 0, verses INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX bible_versions_language ON bible_versions (language);
CREATE TABLE bible_verses (
	version TEXT NOT NULL, book INTEGER NOT NULL, chapter INTEGER NOT NULL, verse INTEGER NOT NULL, text TEXT NOT NULL,
	PRIMARY KEY (version, book, chapter, verse)
) WITHOUT ROWID;
CREATE TABLE bible_read (
	book INTEGER NOT NULL, chapter INTEGER NOT NULL, verse INTEGER NOT NULL, read_at INTEGER NOT NULL,
	PRIMARY KEY (book, chapter, verse)
) WITHOUT ROWID;
CREATE TABLE bible_state (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE bible_daily (
	day TEXT PRIMARY KEY, book INTEGER NOT NULL, chapter INTEGER NOT NULL, verse INTEGER NOT NULL, read INTEGER NOT NULL DEFAULT 0
);
`},
}

// Store implements domain.Library.
type Store struct{ db *sql.DB }

var _ domain.Library = (*Store)(nil)

// New prepares the Bible tables in the database and returns the store.
func New(s *storage.Store) (*Store, error) {
	if err := s.Migrate(Module, migrations); err != nil {
		return nil, err
	}
	return &Store{db: s.DB()}, nil
}

// SaveBooks implements domain.Library.
func (s *Store) SaveBooks(books []domain.Book) error {
	return s.tx(func(tx *sql.Tx) error {
		for _, b := range books {
			names, err := json.Marshal(b.Names)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO bible_books(id, slug, chapters, new_testament, names) VALUES(?, ?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, chapters = excluded.chapters,
				new_testament = excluded.new_testament, names = excluded.names`,
				b.ID, b.Slug, b.Chapters, boolInt(b.New), string(names)); err != nil {
				return err
			}
		}
		return nil
	})
}

// Books implements domain.Library: in the canonical order.
func (s *Store) Books() ([]domain.Book, error) {
	rows, err := s.db.Query(`SELECT id, slug, chapters, new_testament, names FROM bible_books ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Book
	for rows.Next() {
		var b domain.Book
		var isNew int
		var names string
		if err := rows.Scan(&b.ID, &b.Slug, &b.Chapters, &isNew, &names); err != nil {
			return nil, err
		}
		b.New = isNew != 0
		if err := json.Unmarshal([]byte(names), &b.Names); err != nil {
			return nil, fmt.Errorf("sqlstore: names of book %d: %w", b.ID, err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// SaveVersions implements domain.Library: what was downloaded stays downloaded.
func (s *Store) SaveVersions(versions []domain.Version) error {
	return s.tx(func(tx *sql.Tx) error {
		for _, v := range versions {
			if _, err := tx.Exec(`INSERT INTO bible_versions(slug, name, short_name, language, books, chapters) VALUES(?, ?, ?, ?, ?, ?)
				ON CONFLICT(slug) DO UPDATE SET name = excluded.name, short_name = excluded.short_name,
				language = excluded.language, books = excluded.books, chapters = excluded.chapters`,
				v.Slug, v.Name, v.ShortName, v.Language, v.Books, v.Chapters); err != nil {
				return err
			}
		}
		return nil
	})
}

const versionColumns = `slug, name, short_name, language, books, chapters, downloaded, verses`

func scanVersion(sc interface{ Scan(...interface{}) error }) (domain.Version, error) {
	var v domain.Version
	var done int
	err := sc.Scan(&v.Slug, &v.Name, &v.ShortName, &v.Language, &v.Books, &v.Chapters, &done, &v.Verses)
	v.Downloaded = done != 0
	return v, err
}

// Versions implements domain.Library.
func (s *Store) Versions() ([]domain.Version, error) {
	rows, err := s.db.Query(`SELECT ` + versionColumns + ` FROM bible_versions ORDER BY language, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Version
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Version implements domain.Library.
func (s *Store) Version(slug string) (domain.Version, bool, error) {
	v, err := scanVersion(s.db.QueryRow(`SELECT `+versionColumns+` FROM bible_versions WHERE slug = ?`, slug))
	if err == sql.ErrNoRows {
		return domain.Version{}, false, nil
	}
	return v, err == nil, err
}

// SaveChapters implements domain.Library: replaces the verses of the chapters it is given.
func (s *Store) SaveChapters(chapters []domain.ChapterText) error {
	if len(chapters) == 0 {
		return nil
	}
	versions := map[string]bool{}
	err := s.tx(func(tx *sql.Tx) error {
		insert, err := tx.Prepare(`INSERT OR REPLACE INTO bible_verses(version, book, chapter, verse, text) VALUES(?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer insert.Close()
		for _, ch := range chapters {
			versions[ch.Version] = true
			if _, err := tx.Exec(`DELETE FROM bible_verses WHERE version = ? AND book = ? AND chapter = ?`, ch.Version, ch.Book, ch.Chapter); err != nil {
				return err
			}
			for _, v := range ch.Verses {
				if _, err := insert.Exec(ch.Version, ch.Book, ch.Chapter, v.Number, v.Text); err != nil {
					return err
				}
			}
		}
		for version := range versions {
			if _, err := tx.Exec(`UPDATE bible_versions SET verses = (SELECT COUNT(*) FROM bible_verses WHERE version = ?) WHERE slug = ?`, version, version); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// StoredChapters implements domain.Library.
func (s *Store) StoredChapters(version string) (map[domain.ChapterRef]bool, error) {
	rows, err := s.db.Query(`SELECT DISTINCT book, chapter FROM bible_verses WHERE version = ?`, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[domain.ChapterRef]bool{}
	for rows.Next() {
		var r domain.ChapterRef
		if err := rows.Scan(&r.Book, &r.Chapter); err != nil {
			return nil, err
		}
		out[r] = true
	}
	return out, rows.Err()
}

// MarkDownloaded implements domain.Library.
func (s *Store) MarkDownloaded(version string) error {
	_, err := s.db.Exec(`UPDATE bible_versions SET downloaded = 1,
		verses = (SELECT COUNT(*) FROM bible_verses WHERE version = ?) WHERE slug = ?`, version, version)
	return err
}

// DeleteVersion implements domain.Library: frees the space of the text, keeps the catalog row.
func (s *Store) DeleteVersion(version string) error {
	return s.tx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM bible_verses WHERE version = ?`, version); err != nil {
			return err
		}
		_, err := tx.Exec(`UPDATE bible_versions SET downloaded = 0, verses = 0 WHERE slug = ?`, version)
		return err
	})
}

// Chapter implements domain.Library: one range read, a chapter at most.
func (s *Store) Chapter(version string, book, chapter int) (domain.ChapterText, error) {
	out := domain.ChapterText{Version: version, Book: book, Chapter: chapter}
	rows, err := s.db.Query(`SELECT verse, text FROM bible_verses WHERE version = ? AND book = ? AND chapter = ? ORDER BY verse`, version, book, chapter)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domain.Verse
		if err := rows.Scan(&v.Number, &v.Text); err != nil {
			return out, err
		}
		out.Verses = append(out.Verses, v)
	}
	return out, rows.Err()
}

// Previews implements domain.Library.
func (s *Store) Previews(version string, book, max int) (map[int]string, error) {
	rows, err := s.db.Query(`SELECT chapter, substr(text, 1, ?) FROM bible_verses WHERE version = ? AND book = ? AND verse = 1`, max, version, book)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var chapter int
		var text string
		if err := rows.Scan(&chapter, &text); err != nil {
			return nil, err
		}
		out[chapter] = text
	}
	return out, rows.Err()
}

// VerseText implements domain.Library.
func (s *Store) VerseText(version string, ref domain.Ref) (string, bool, error) {
	var text string
	err := s.db.QueryRow(`SELECT text FROM bible_verses WHERE version = ? AND book = ? AND chapter = ? AND verse = ?`,
		version, ref.Book, ref.Chapter, ref.Verse).Scan(&text)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return text, err == nil, err
}

// Search implements domain.Library: the rows are read one at a time and dropped, so the
// memory used does not depend on the size of the version.
func (s *Store) Search(version string, words []string, limit int) ([]domain.Hit, error) {
	if len(words) == 0 || limit < 1 {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT book, chapter, verse, text FROM bible_verses WHERE version = ?`, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Hit
	for rows.Next() {
		var h domain.Hit
		if err := rows.Scan(&h.Ref.Book, &h.Ref.Chapter, &h.Ref.Verse, &h.Text); err != nil {
			return nil, err
		}
		if domain.MatchesAll(h.Text, words) {
			out = append(out, h)
			if len(out) == limit {
				break
			}
		}
	}
	return out, rows.Err()
}

// PickVerse implements domain.Library.
func (s *Store) PickVerse(version string, n int) (domain.Ref, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM bible_verses WHERE version = ?`, version).Scan(&total); err != nil {
		return domain.Ref{}, err
	}
	if total == 0 {
		return domain.Ref{}, nil
	}
	if n < 0 {
		n = -n
	}
	var r domain.Ref
	err := s.db.QueryRow(`SELECT book, chapter, verse FROM bible_verses WHERE version = ? ORDER BY book, chapter, verse LIMIT 1 OFFSET ?`,
		version, n%total).Scan(&r.Book, &r.Chapter, &r.Verse)
	return r, err
}

// MarkRead implements domain.Library: a verse read again keeps its first date.
func (s *Store) MarkRead(refs []domain.Ref, at time.Time) error {
	if len(refs) == 0 {
		return nil
	}
	return s.tx(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`INSERT OR IGNORE INTO bible_read(book, chapter, verse, read_at) VALUES(?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, r := range refs {
			if _, err := stmt.Exec(r.Book, r.Chapter, r.Verse, at.Unix()); err != nil {
				return err
			}
		}
		return nil
	})
}

// IsRead implements domain.Library.
func (s *Store) IsRead(ref domain.Ref) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM bible_read WHERE book = ? AND chapter = ? AND verse = ?`, ref.Book, ref.Chapter, ref.Verse).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// ReadCount implements domain.Library. Only the verses the version has count, on both
// sides: a version with fewer books (or more) changes the total, and the verses read
// in another version that this one lacks do not.
func (s *Store) ReadCount(version string) (read, total int, err error) {
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM bible_verses WHERE version = ?`, version).Scan(&total); err != nil {
		return 0, 0, err
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM bible_read r JOIN bible_verses v
		ON v.version = ? AND v.book = r.book AND v.chapter = r.chapter AND v.verse = r.verse`, version).Scan(&read)
	return read, total, err
}

const (
	keyPosition = "position"
	keyPrefs    = "preferences"
)

func (s *Store) putState(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bible_state(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, string(data))
	return err
}

func (s *Store) getState(key string, into interface{}) (bool, error) {
	var data string
	err := s.db.QueryRow(`SELECT value FROM bible_state WHERE key = ?`, key).Scan(&data)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(data), into)
}

// SavePosition implements domain.Library.
func (s *Store) SavePosition(p domain.Position) error { return s.putState(keyPosition, p) }

// Position implements domain.Library: the zero Position when nothing was opened yet.
func (s *Store) Position() (domain.Position, error) {
	var p domain.Position
	_, err := s.getState(keyPosition, &p)
	return p, err
}

// LoadPreferences implements domain.Library: false when nothing was saved yet.
func (s *Store) LoadPreferences() (domain.Preferences, bool, error) {
	p := domain.DefaultPreferences()
	ok, err := s.getState(keyPrefs, &p)
	return p.Normalize(), ok, err
}

// SavePreferences implements domain.Library.
func (s *Store) SavePreferences(p domain.Preferences) error {
	return s.putState(keyPrefs, p.Normalize())
}

// Daily implements domain.Library.
func (s *Store) Daily(day string) (domain.DailyVerse, bool, error) {
	d := domain.DailyVerse{Day: day}
	var read int
	err := s.db.QueryRow(`SELECT book, chapter, verse, read FROM bible_daily WHERE day = ?`, day).Scan(&d.Ref.Book, &d.Ref.Chapter, &d.Ref.Verse, &read)
	if err == sql.ErrNoRows {
		return d, false, nil
	}
	d.Read = read != 0
	return d, err == nil, err
}

// SaveDaily implements domain.Library: it also forgets the days older than a month.
func (s *Store) SaveDaily(d domain.DailyVerse) error {
	return s.tx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO bible_daily(day, book, chapter, verse, read) VALUES(?, ?, ?, ?, ?)
			ON CONFLICT(day) DO UPDATE SET book = excluded.book, chapter = excluded.chapter, verse = excluded.verse, read = excluded.read`,
			d.Day, d.Ref.Book, d.Ref.Chapter, d.Ref.Verse, boolInt(d.Read)); err != nil {
			return err
		}
		if t, err := time.Parse("2006-01-02", d.Day); err == nil {
			_, err = tx.Exec(`DELETE FROM bible_daily WHERE day < ?`, t.AddDate(0, 0, -31).Format("2006-01-02"))
			return err
		}
		return nil
	})
}

func (s *Store) tx(f func(*sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := f(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
