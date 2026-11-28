package hub

import (
	"database/sql"
	"fmt"
	"time"
)

// migration is one schema step. Set sql, fn or both (sql runs first).
// Versions are consecutive integers starting at 1. Never edit or reorder a
// migration that has been released: append a new one.
type migration struct {
	version int
	name    string
	sql     string
	fn      func(tx *sql.Tx) error
}

// migrations is the ordered list applied by OpenStore.
var migrations = []migration{
	{version: 1, name: "initial schema", sql: schemaV1},
}

const schemaV1 = `
CREATE TABLE IF NOT EXISTS servers (
  name TEXT PRIMARY KEY, hello TEXT NOT NULL, last_seen INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS sys_metrics (
  server TEXT NOT NULL, ts INTEGER NOT NULL, cpu REAL, mem_pct REAL, load1 REAL,
  disk_pct REAL, rx REAL, tx REAL);
CREATE INDEX IF NOT EXISTS sys_metrics_idx ON sys_metrics(server, ts);
CREATE TABLE IF NOT EXISTS proc_metrics (
  server TEXT NOT NULL, name TEXT NOT NULL, inst INTEGER NOT NULL, ts INTEGER NOT NULL,
  cpu REAL, mem INTEGER, restarts INTEGER, status TEXT);
CREATE INDEX IF NOT EXISTS proc_metrics_idx ON proc_metrics(server, name, ts);
CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY AUTOINCREMENT, ts INTEGER NOT NULL, server TEXT NOT NULL,
  name TEXT NOT NULL, inst INTEGER NOT NULL, type TEXT NOT NULL, status TEXT,
  exit_code INTEGER, reason TEXT, manual INTEGER NOT NULL DEFAULT 0);
CREATE INDEX IF NOT EXISTS events_ts ON events(ts);
CREATE TABLE IF NOT EXISTS issues (
  fp TEXT NOT NULL, server TEXT NOT NULL, name TEXT NOT NULL, sample TEXT NOT NULL,
  count INTEGER NOT NULL, first_seen INTEGER NOT NULL, last_seen INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'open', PRIMARY KEY (fp, server, name));
CREATE TABLE IF NOT EXISTS alert_rules (id INTEGER PRIMARY KEY AUTOINCREMENT, body TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS alert_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT, ts INTEGER NOT NULL, rule_id INTEGER, rule_name TEXT,
  level TEXT, server TEXT, proc TEXT, state TEXT, message TEXT, value REAL, notified INTEGER);
CREATE INDEX IF NOT EXISTS alert_history_ts ON alert_history(ts);
CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
`

// migrate applies every migration newer than the recorded version, each in
// its own transaction together with its schema_migrations row.
func migrate(db *sql.DB, list []migration) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY, name TEXT, applied_at INTEGER)`); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	var cur int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&cur); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	latest := 0
	if n := len(list); n > 0 {
		latest = list[n-1].version
	}
	if cur > latest {
		return fmt.Errorf("migrations: database schema version %d is newer than this binary supports (%d); upgrade pm2go", cur, latest)
	}
	if cur == 0 && len(list) > 0 {
		// Database created before versioning existed: tables are there, no
		// version rows. Record version 1 without running it.
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='servers'`).Scan(&n); err != nil {
			return fmt.Errorf("migrations: %w", err)
		}
		if n > 0 {
			if _, err := db.Exec(`INSERT INTO schema_migrations(version,name,applied_at) VALUES(?,?,?)`,
				list[0].version, list[0].name, time.Now().UnixMilli()); err != nil {
				return fmt.Errorf("migration %d (%s): %w", list[0].version, list[0].name, err)
			}
			cur = list[0].version
		}
	}
	for _, m := range list {
		if m.version <= cur {
			continue
		}
		if err := applyMigration(db, m); err != nil {
			return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
		}
	}
	return nil
}

func applyMigration(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if m.sql != "" {
		if _, err := tx.Exec(m.sql); err != nil {
			return err
		}
	}
	if m.fn != nil {
		if err := m.fn(tx); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations(version,name,applied_at) VALUES(?,?,?)`,
		m.version, m.name, time.Now().UnixMilli()); err != nil {
		return err
	}
	return tx.Commit()
}
