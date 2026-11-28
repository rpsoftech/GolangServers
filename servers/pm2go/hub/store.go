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

func OpenStore(home string) (*Store, error) {
	dsn := "file:" + filepath.Join(home, "hub.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: one writer, keeps it simple and race-free
	if err := migrate(db, migrations); err != nil {
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
