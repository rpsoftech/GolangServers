package hub

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database.
type Store struct{ db *sql.DB }

const schema = `
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

func OpenStore(home string) (*Store, error) {
	dsn := "file:" + filepath.Join(home, "hub.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: one writer, keeps it simple and race-free
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) GetSetting(key string, out any) (bool, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(v), out)
}

func (s *Store) PutSetting(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, string(b))
	return err
}

func (s *Store) Prune(retention time.Duration) error {
	cut := time.Now().Add(-retention).UnixMilli()
	for _, q := range []string{
		`DELETE FROM sys_metrics WHERE ts<?`, `DELETE FROM proc_metrics WHERE ts<?`,
		`DELETE FROM events WHERE ts<?`, `DELETE FROM alert_history WHERE ts<?`,
	} {
		if _, err := s.db.Exec(q, cut); err != nil {
			return fmt.Errorf("prune: %w", err)
		}
	}
	return nil
}
