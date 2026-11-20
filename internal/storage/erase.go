package storage

import (
	"fmt"
	"os"
	"strings"
)

// DropModule deletes every table of a module (the tables named "<module>_...") and forgets its
// schema version, all or nothing, then gives the freed space back to the file. It is what removing
// an app does to the database it shared with the other apps: their tables stay.
func (s *Store) DropModule(module string) error {
	if !identifier.MatchString(module) {
		return fmt.Errorf("storage: %q is not a module name", module)
	}
	tables, err := s.Tables()
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, t := range tables {
		if strings.HasPrefix(t, module+"_") {
			if _, err := tx.Exec("DROP TABLE IF EXISTS " + t); err != nil {
				tx.Rollback()
				return fmt.Errorf("storage: dropping %s: %w", t, err)
			}
		}
	}
	if _, err := tx.Exec(`DELETE FROM schema_versions WHERE module = ?`, module); err != nil && !strings.Contains(err.Error(), "no such table") {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, _ = s.db.Exec(`VACUUM`) // a courtesy: the data is gone either way
	return nil
}

// EraseModule drops a module's tables from the database file at path (nothing happens when the
// file does not exist) and deletes the file itself when no other module has tables left in it.
func EraseModule(path, module string) error {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	s, err := Open(path)
	if err != nil {
		return err
	}
	if err := s.DropModule(module); err != nil {
		s.Close()
		return err
	}
	left, err := s.Tables()
	s.Close()
	if err != nil {
		return err
	}
	for _, t := range left {
		if t != "schema_versions" {
			return nil
		}
	}
	for _, f := range []string{path, path + "-journal"} {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
