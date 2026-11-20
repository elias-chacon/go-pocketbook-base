package storage

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func open(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

var mailMigrations = []Migration{
	{1, `CREATE TABLE messages (id TEXT PRIMARY KEY, account_id TEXT NOT NULL, subject TEXT)`},
	{2, `CREATE INDEX messages_account ON messages(account_id)`},
}

func TestTheDatabaseKeepsDataAcrossOpens(t *testing.T) {
	s, path := open(t)
	if err := s.Migrate("mail", mailMigrations); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB().Exec(`INSERT INTO messages VALUES ('m1', 'a1', 'Olá')`); err != nil {
		t.Fatal(err)
	}
	s.Close()

	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	var subject string
	if err := again.DB().QueryRow(`SELECT subject FROM messages WHERE id = 'm1'`).Scan(&subject); err != nil || subject != "Olá" {
		t.Errorf("%q %v", subject, err)
	}
	if err := again.Check(); err != nil {
		t.Error(err)
	}
}

func TestMigrationsRunOnceInOrderAndAreSafeToRepeat(t *testing.T) {
	s, _ := open(t)
	if v, err := s.Version("mail"); err != nil || v != 0 {
		t.Fatalf("never migrated: %d %v", v, err)
	}
	if err := s.Migrate("mail", mailMigrations[:1]); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Version("mail"); v != 1 {
		t.Errorf("version %d", v)
	}
	// a new release brings step 2; step 1 does not run again (it would fail: the table exists)
	if err := s.Migrate("mail", mailMigrations); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate("mail", mailMigrations); err != nil {
		t.Errorf("repeating is harmless: %v", err)
	}
	if v, _ := s.Version("mail"); v != 2 {
		t.Errorf("version %d", v)
	}
	// the order given does not matter: it is the version number
	other := []Migration{{2, `CREATE TABLE b (x)`}, {1, `CREATE TABLE a (x)`}}
	if err := s.Migrate("calendar", other); err != nil {
		t.Fatal(err)
	}
	tables, _ := s.Tables()
	if !reflect.DeepEqual(tables, []string{"a", "b", "messages", "schema_versions"}) {
		t.Errorf("%v", tables)
	}
}

func TestAFailingMigrationLeavesNothingHalfDone(t *testing.T) {
	s, _ := open(t)
	bad := []Migration{
		{1, `CREATE TABLE good (x)`},
		{2, `CREATE TABLE half (x); CREATE TABLE half (x)`}, // the second statement fails
	}
	err := s.Migrate("mod", bad)
	if err == nil || !strings.Contains(err.Error(), "migration 2 of mod") {
		t.Fatalf("%v", err)
	}
	if v, _ := s.Version("mod"); v != 1 {
		t.Errorf("only step 1 was recorded: %d", v)
	}
	tables, _ := s.Tables()
	for _, tb := range tables {
		if tb == "half" {
			t.Error("a failed step is rolled back entirely")
		}
	}
}

func TestMigrationsAreCheckedBeforeAnythingRuns(t *testing.T) {
	s, _ := open(t)
	if err := s.Migrate("Bad Name", nil); err == nil {
		t.Error("a module name is an identifier")
	}
	if err := s.Migrate("mod", []Migration{{1, `CREATE TABLE a (x)`}, {3, `CREATE TABLE c (x)`}}); err == nil || !strings.Contains(err.Error(), "gaps") {
		t.Errorf("a gap: %v", err)
	}
	if err := s.Migrate("mod", []Migration{{2, `CREATE TABLE a (x)`}}); err == nil {
		t.Error("versions start at 1")
	}
	if tables, _ := s.Tables(); len(tables) != 0 {
		t.Errorf("nothing ran: %v", tables)
	}
}

func TestADatabaseFromANewerProgramIsRefused(t *testing.T) {
	s, _ := open(t)
	s.Migrate("mail", mailMigrations)
	if err := s.Migrate("mail", mailMigrations[:1]); err != ErrDowngrade {
		t.Errorf("an older program must not touch a newer schema: %v", err)
	}
}

func TestErasingAnAccountClearsItsTablesAndOnlyThem(t *testing.T) {
	s, _ := open(t)
	s.Migrate("mail", mailMigrations)
	s.Migrate("calendar", []Migration{{1, `CREATE TABLE events (id TEXT PRIMARY KEY, account_id TEXT NOT NULL, title TEXT)`}})
	for _, table := range []string{"messages", "events", "messages"} { // twice: harmless
		if err := s.RegisterAccountTable(table); err != nil {
			t.Fatal(err)
		}
	}
	db := s.DB()
	db.Exec(`INSERT INTO messages VALUES ('m1','ana','x'), ('m2','bia','y')`)
	db.Exec(`INSERT INTO events VALUES ('e1','ana','a'), ('e2','bia','b')`)

	if err := s.Erase("ana"); err != nil {
		t.Fatal(err)
	}
	count := func(table string) int {
		var n int
		db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
		return n
	}
	if count("messages") != 1 || count("events") != 1 {
		t.Errorf("only ana's rows are gone: %d %d", count("messages"), count("events"))
	}
	if err := s.EraseAccount("nobody"); err != nil {
		t.Errorf("erasing an unknown account is harmless: %v", err)
	}
	if err := s.RegisterAccountTable("drop table;"); err == nil {
		t.Error("a table name is an identifier")
	}
}

func TestErasingStopsAndRollsBackWhenATableIsMissing(t *testing.T) {
	s, _ := open(t)
	s.Migrate("mail", mailMigrations)
	s.RegisterAccountTable("messages")
	s.RegisterAccountTable("no_such_table")
	s.DB().Exec(`INSERT INTO messages VALUES ('m1','ana','x')`)
	if err := s.EraseAccount("ana"); err == nil || !strings.Contains(err.Error(), "no_such_table") {
		t.Fatalf("%v", err)
	}
	var n int
	s.DB().QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&n)
	if n != 1 {
		t.Error("all or nothing: the first table was not left half-cleared")
	}
}

func TestOpenReportsAFileThatCannotBeOpened(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "no-such-dir", "data.db")); err == nil {
		t.Error("a folder that does not exist")
	}
	junk := filepath.Join(t.TempDir(), "junk.db")
	os.WriteFile(junk, []byte(strings.Repeat("not a database ", 100)), 0o644)
	s, err := Open(junk)
	if err == nil {
		defer s.Close()
		if s.Check() == nil {
			t.Error("a file that is not a database must not look sound")
		}
	}
}

func TestMemoryDatabasesWorkForTests(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Migrate("mail", mailMigrations); err != nil {
		t.Fatal(err)
	}
	var journal string
	s.DB().QueryRow(`PRAGMA foreign_keys`).Scan(&journal)
	if journal != "1" {
		t.Errorf("foreign keys are on: %q", journal)
	}
}

func TestEraseModuleDropsOnlyThatModulesTablesAndDeletesTheFileWhenNoneIsLeft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "productivity.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"tasks", "mail"} {
		if err := s.Migrate(m, []Migration{{Version: 1, SQL: "CREATE TABLE " + m + "_items (id TEXT)"}, {Version: 2, SQL: "CREATE TABLE " + m + "_more (id TEXT)"}}); err != nil {
			t.Fatal(err)
		}
	}
	s.Close()

	if err := EraseModule(path, "tasks"); err != nil {
		t.Fatal(err)
	}
	s, _ = Open(path)
	tables, _ := s.Tables()
	if strings.Join(tables, ",") != "mail_items,mail_more,schema_versions" {
		t.Errorf("only the tasks tables go: %v", tables)
	}
	if v, _ := s.Version("tasks"); v != 0 {
		t.Errorf("the schema version of tasks is forgotten: %d", v)
	}
	s.Close()

	if err := EraseModule(path, "mail"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("with no module left the file is deleted")
	}
	if err := EraseModule(path, "mail"); err != nil {
		t.Errorf("a file that is not there is not a failure: %v", err)
	}
}
