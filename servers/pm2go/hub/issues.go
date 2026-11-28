package hub

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Issue groups similar error lines, like the PM2 Plus issues dashboard.
type Issue struct {
	Fingerprint string `json:"fingerprint"`
	Server      string `json:"server"`
	Name        string `json:"name"`
	Sample      string `json:"sample"`
	Count       int64  `json:"count"`
	FirstSeen   int64  `json:"first_seen"`
	LastSeen    int64  `json:"last_seen"`
	Status      string `json:"status"` // open, resolved, ignored
	dirty       bool
}

var (
	errorLine = regexp.MustCompile(`(?i)\b(error|fatal|panic|exception|traceback|uncaught|unhandled|segfault)\b`)
	reNumber  = regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	reHex     = regexp.MustCompile(`\b(0x)?[0-9a-f]{8,}\b`)
	reQuoted  = regexp.MustCompile(`"[^"]*"|'[^']*'`)
	reTime    = regexp.MustCompile(`\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?`)
)

// IsErrorLine reports whether a line looks like an error.
func IsErrorLine(line string) bool { return errorLine.MatchString(line) }

// Fingerprint reduces a line to a stable shape: timestamps, ids, numbers and
// quoted values are replaced so repeated occurrences group together.
func Fingerprint(line string) string {
	s := reTime.ReplaceAllString(line, "<t>")
	s = reQuoted.ReplaceAllString(s, "<q>")
	s = reHex.ReplaceAllString(s, "<h>")
	s = reNumber.ReplaceAllString(s, "<n>")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 300 {
		s = s[:300]
	}
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:8])
}

type Issues struct {
	store *Store
	mu    sync.Mutex
	byKey map[string]*Issue
}

func NewIssues(store *Store) *Issues {
	is := &Issues{store: store, byKey: map[string]*Issue{}}
	rows, err := store.db.Query(`SELECT fp,server,name,sample,count,first_seen,last_seen,status FROM issues`)
	if err != nil {
		return is
	}
	defer rows.Close()
	for rows.Next() {
		var i Issue
		if rows.Scan(&i.Fingerprint, &i.Server, &i.Name, &i.Sample, &i.Count, &i.FirstSeen, &i.LastSeen, &i.Status) == nil {
			is.byKey[issueKey(i.Server, i.Name, i.Fingerprint)] = &i
		}
	}
	return is
}

func issueKey(server, name, fp string) string { return server + "|" + name + "|" + fp }

// Observe records the line if it looks like an error.
func (is *Issues) Observe(server, name, line string, ts int64) {
	if !IsErrorLine(line) {
		return
	}
	fp := Fingerprint(line)
	k := issueKey(server, name, fp)
	is.mu.Lock()
	defer is.mu.Unlock()
	i := is.byKey[k]
	if i == nil {
		sample := line
		if len(sample) > 500 {
			sample = sample[:500]
		}
		i = &Issue{Fingerprint: fp, Server: server, Name: name, Sample: sample, FirstSeen: ts, Status: "open"}
		is.byKey[k] = i
	}
	i.Count++
	i.LastSeen = ts
	if i.Status == "resolved" { // came back
		i.Status = "open"
	}
	i.dirty = true
}

// List returns issues, newest activity first. status "" means all.
func (is *Issues) List(status string) []Issue {
	is.mu.Lock()
	defer is.mu.Unlock()
	out := make([]Issue, 0, len(is.byKey))
	for _, i := range is.byKey {
		if status == "" || i.Status == status {
			out = append(out, *i)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].LastSeen > out[b].LastSeen })
	return out
}

func (is *Issues) SetStatus(server, name, fp, status string) bool {
	is.mu.Lock()
	defer is.mu.Unlock()
	i := is.byKey[issueKey(server, name, fp)]
	if i == nil {
		return false
	}
	i.Status, i.dirty = status, true
	return true
}

// Flush writes changed issues to the database.
func (is *Issues) Flush() {
	is.mu.Lock()
	var dirty []Issue
	for _, i := range is.byKey {
		if i.dirty {
			dirty = append(dirty, *i)
			i.dirty = false
		}
	}
	is.mu.Unlock()
	for _, i := range dirty {
		_, err := is.store.db.Exec(`INSERT INTO issues(fp,server,name,sample,count,first_seen,last_seen,status) VALUES(?,?,?,?,?,?,?,?)
			ON CONFLICT(fp,server,name) DO UPDATE SET count=excluded.count,last_seen=excluded.last_seen,status=excluded.status`,
			i.Fingerprint, i.Server, i.Name, i.Sample, i.Count, i.FirstSeen, i.LastSeen, i.Status)
		if err != nil {
			log.Printf("flush issue: %v", err)
		}
	}
}

func (is *Issues) RunFlusher(stop <-chan struct{}) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			is.Flush()
			return
		case <-t.C:
			is.Flush()
		}
	}
}
