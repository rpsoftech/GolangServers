package hub

import (
	"log"
	"strings"
	"time"
)

// AuditEntry is one row of the audit log. Detail never holds secrets: no
// passwords, tokens, bot tokens or session ids, only names, ids and flags.
type AuditEntry struct {
	ID      int64  `json:"id"`
	TS      int64  `json:"ts"`
	User    string `json:"user"`
	IP      string `json:"ip"`
	Action  string `json:"action"`
	Target  string `json:"target"`
	Detail  string `json:"detail"`
	Success bool   `json:"success"`
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Audit appends a row. Failures are logged, never returned: an audit write
// must not change the outcome of the action it records.
func (s *Store) Audit(e AuditEntry) {
	if e.TS == 0 {
		e.TS = time.Now().UnixMilli()
	}
	_, err := s.db.Exec(`INSERT INTO audit(ts,user,ip,action,target,detail,success) VALUES(?,?,?,?,?,?,?)`,
		e.TS, clip(e.User, 100), clip(e.IP, 64), e.Action, clip(e.Target, 200), clip(e.Detail, 500), b2i(e.Success))
	if err != nil {
		log.Printf("audit write failed: %v", err)
	}
}

// AuditFilter narrows ListAudit. Zero values mean no filter.
type AuditFilter struct {
	User, Action string
	OnlyFailed   bool
	Before       int64 // return rows with id < Before (paging)
	Limit        int
}

func (s *Store) ListAudit(f AuditFilter) ([]AuditEntry, error) {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	var where []string
	var args []any
	if f.User != "" {
		where, args = append(where, "user=?"), append(args, f.User)
	}
	if f.Action != "" {
		if strings.HasSuffix(f.Action, "*") { // prefix match: "user.*"
			where, args = append(where, "action LIKE ?"), append(args, strings.TrimSuffix(f.Action, "*")+"%")
		} else {
			where, args = append(where, "action=?"), append(args, f.Action)
		}
	}
	if f.OnlyFailed {
		where = append(where, "success=0")
	}
	if f.Before > 0 {
		where, args = append(where, "id<?"), append(args, f.Before)
	}
	q := `SELECT id,ts,user,ip,action,target,detail,success FROM audit`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	rows, err := s.db.Query(q+` ORDER BY id DESC LIMIT ?`, append(args, f.Limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		var ok int
		if rows.Scan(&e.ID, &e.TS, &e.User, &e.IP, &e.Action, &e.Target, &e.Detail, &ok) == nil {
			e.Success = ok != 0
			out = append(out, e)
		}
	}
	return out, rows.Err()
}
