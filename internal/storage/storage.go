// Package storage is the shared local database of the productivity apps: one
// SQLite file (applications/.pbgo/data.db) holding the cached messages, events and
// tasks, so the screens read from the reader's own disk and never wait for the
// network (offline-first).
//
// It is pure Go (modernc.org/sqlite), so it builds with the SDK's Go 1.17 and the
// tests run on any computer without a C compiler. Each context keeps its own
// tables and its own migrations (see Migrate); the database knows nothing of
// messages or events, only how to open, version, and clean itself.
//
// The settings are the ones a small reader needs: a small page cache, a rollback
// journal that is truncated, not deleted (the reader's storage is a FAT volume,
// where the write-ahead log and its shared memory are not safe), and foreign keys.
package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"

	_ "modernc.org/sqlite" // the driver, registered as "sqlite"
)

// Store is an open database.
type Store struct {
	db *sql.DB

	mu            sync.Mutex
	accountTables []string
}

// pragmas are applied to every connection.
var pragmas = []string{
	"journal_mode(TRUNCATE)", // not WAL: see the package comment
	"synchronous(NORMAL)",
	"cache_size(-2000)", // about 2 MB of pages: the reader has 1 GB for everything
	"foreign_keys(1)",
	"busy_timeout(5000)",
}

// Open opens (and creates, if needed) the database file at path. ":memory:" gives a
// database that lives only in memory, for tests.
func Open(path string) (*Store, error) {
	q := url.Values{}
	for _, p := range pragmas {
		q.Add("_pragma", p)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	// one connection: a single app runs at a time, and an in-memory database is per connection
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: cannot open %s: %w", path, err)
	}
	return &Store{db: db}, nil
}

// DB is the connection pool, for the contexts to run their queries on.
func (s *Store) DB() *sql.DB { return s.db }

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Migration is one step of a module's schema: SQL that takes the schema from
// Version-1 to Version. Versions start at 1 and have no gaps.
type Migration struct {
	Version int
	SQL     string
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// ErrDowngrade is returned when the database was written by a newer version of the
// software than the one running (it knows migrations the code does not).
var ErrDowngrade = errors.New("storage: the database is newer than this program")

// Migrate brings a module's tables to the latest version: the steps not applied yet
// run, in order, each in its own transaction, and the version is recorded with it.
// It is safe to call at every start. module names the owner ("mail", "calendar").
func (s *Store) Migrate(module string, migrations []Migration) error {
	if !identifier.MatchString(module) {
		return fmt.Errorf("storage: %q is not a module name", module)
	}
	sorted := append([]Migration(nil), migrations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Version < sorted[j].Version })
	for i, m := range sorted {
		if m.Version != i+1 {
			return fmt.Errorf("storage: the migrations of %s must be numbered 1, 2, 3... without gaps (found %d at position %d)", module, m.Version, i+1)
		}
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_versions (module TEXT PRIMARY KEY, version INTEGER NOT NULL)`); err != nil {
		return err
	}
	current, err := s.version(module)
	if err != nil {
		return err
	}
	if current > len(sorted) {
		return ErrDowngrade
	}
	for _, m := range sorted[current:] {
		if err := s.apply(module, m); err != nil {
			return fmt.Errorf("storage: migration %d of %s: %w", m.Version, module, err)
		}
	}
	return nil
}

func (s *Store) version(module string) (int, error) {
	var v int
	err := s.db.QueryRow(`SELECT version FROM schema_versions WHERE module = ?`, module).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return v, err
}

func (s *Store) apply(module string, m Migration) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(m.SQL); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_versions(module, version) VALUES(?, ?)
		ON CONFLICT(module) DO UPDATE SET version = excluded.version`, module, m.Version); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Version is the schema version of a module (0 when it was never migrated).
func (s *Store) Version(module string) (int, error) {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_versions (module TEXT PRIMARY KEY, version INTEGER NOT NULL)`); err != nil {
		return 0, err
	}
	return s.version(module)
}

// RegisterAccountTable declares that a table has an account_id column: EraseAccount
// clears it. The contexts register their tables after migrating them.
func (s *Store) RegisterAccountTable(table string) error {
	if !identifier.MatchString(table) {
		return fmt.Errorf("storage: %q is not a table name", table)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.accountTables {
		if t == table {
			return nil
		}
	}
	s.accountTables = append(s.accountTables, table)
	return nil
}

// EraseAccount deletes everything the account left in the registered tables, all
// or nothing. It is the Accounts context's Eraser: removing an account removes its
// messages, events and tasks from the reader.
func (s *Store) EraseAccount(accountID string) error {
	s.mu.Lock()
	tables := append([]string(nil), s.accountTables...)
	s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, table := range tables {
		if _, err := tx.Exec("DELETE FROM "+table+" WHERE account_id = ?", accountID); err != nil {
			tx.Rollback()
			return fmt.Errorf("storage: clearing %s: %w", table, err)
		}
	}
	return tx.Commit()
}

// Erase implements the Accounts context's Eraser port.
func (s *Store) Erase(accountID string) error { return s.EraseAccount(accountID) }

// Tables lists the user tables of the database (for diagnostics and tests).
func (s *Store) Tables() ([]string, error) {
	rows, err := s.db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// Check runs SQLite's integrity check: nil when the file is sound.
func (s *Store) Check() error {
	var result string
	if err := s.db.QueryRow(`PRAGMA integrity_check`).Scan(&result); err != nil {
		return err
	}
	if !strings.EqualFold(result, "ok") {
		return fmt.Errorf("storage: the database is damaged: %s", result)
	}
	return nil
}
