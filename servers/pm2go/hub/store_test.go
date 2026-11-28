package hub

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func rawDB(t *testing.T, home string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(home, "hub.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func versions(t *testing.T, db *sql.DB) []int {
	t.Helper()
	rows, err := db.Query(`SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var v int
		rows.Scan(&v)
		out = append(out, v)
	}
	return out
}

func TestMigrateFresh(t *testing.T) {
	home := t.TempDir()
	s, err := OpenStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if got := versions(t, s.db); len(got) != len(migrations) || got[0] != 1 {
		t.Fatalf("versions = %v", got)
	}
	if err := s.PutSetting("k", "v"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	// idempotent reopen: nothing applied, data kept
	s, err = OpenStore(home)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got := versions(t, s.db); len(got) != len(migrations) {
		t.Fatalf("versions after reopen = %v", got)
	}
	var v string
	if ok, _ := s.GetSetting("k", &v); !ok || v != "v" {
		t.Fatalf("setting lost: %v %q", ok, v)
	}
}

func TestMigrateAdoptsExisting(t *testing.T) {
	home := t.TempDir()
	db := rawDB(t, home)
	if _, err := db.Exec(schemaV1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO servers(name,hello,last_seen) VALUES('box','{}',5)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := OpenStore(home)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got := versions(t, s.db); len(got) != len(migrations) || got[0] != 1 {
		t.Fatalf("versions = %v", got)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM servers WHERE name='box'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("row lost: %d %v", n, err)
	}
}

func TestMigrateFailureRollsBack(t *testing.T) {
	db := rawDB(t, t.TempDir())
	list := []migration{
		{version: 1, name: "one", sql: `CREATE TABLE a (x INTEGER);`},
		{version: 2, name: "two", sql: `CREATE TABLE b (x INTEGER);`, fn: func(tx *sql.Tx) error {
			_, err := tx.Exec(`INSERT INTO nope VALUES (1)`)
			return err
		}},
	}
	err := migrate(db, list)
	if err == nil || !strings.Contains(err.Error(), "migration 2") {
		t.Fatalf("err = %v", err)
	}
	if got := versions(t, db); len(got) != 1 || got[0] != 1 {
		t.Fatalf("versions = %v", got)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='b'`).Scan(&n)
	if n != 0 {
		t.Fatal("table b survived rollback")
	}
}

func TestMigrateFuncAndUpgrade(t *testing.T) {
	db := rawDB(t, t.TempDir())
	list := []migration{{version: 1, name: "one", sql: `CREATE TABLE a (x INTEGER);`}}
	if err := migrate(db, list); err != nil {
		t.Fatal(err)
	}
	list = append(list, migration{version: 2, name: "fix", fn: func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO a VALUES (7)`)
		return err
	}})
	for i := 0; i < 2; i++ { // second run applies nothing
		if err := migrate(db, list); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM a`).Scan(&n)
	if n != 1 {
		t.Fatalf("rows = %d, want 1", n)
	}
}

func TestMigrateRefusesNewerDatabase(t *testing.T) {
	home := t.TempDir()
	s, err := OpenStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO schema_migrations(version,name,applied_at) VALUES(999,'future',0)`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	_, err = OpenStore(home)
	if err == nil || !strings.Contains(err.Error(), "newer than this binary") {
		t.Fatalf("err = %v", err)
	}
}
