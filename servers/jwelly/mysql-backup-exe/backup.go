package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

type DestResult struct {
	Label string
	Err   error
}

type Result struct {
	Name     string
	Engine   string
	Size     int64
	Duration time.Duration
	// Err is a dump failure; destination failures are in Dests.
	Err   error
	Dests []DestResult
}

func (r *Result) OK() bool {
	if r.Err != nil {
		return false
	}
	for _, d := range r.Dests {
		if d.Err != nil {
			return false
		}
	}
	return true
}

// RunBackup dumps one database and delivers the file to every destination. A
// failing destination does not stop the others. notify is called for problems
// the user must act on (an unusable local folder).
func RunBackup(ctx context.Context, db *DatabaseConfig, notify Notifier) *Result {
	start := time.Now()
	res := &Result{Name: db.Name}
	defer func() { res.Duration = time.Since(start) }()

	tmp, err := os.MkdirTemp("", "mysql-backup-*")
	if err != nil {
		res.Err = err
		return res
	}
	defer os.RemoveAll(tmp)

	log.Printf("[%s] dumping %s@%s...", db.Name, db.MysqlConfig.Database, db.MysqlConfig.Host)
	file, engine, err := DumpToFile(ctx, db, tmp)
	if err != nil {
		res.Err = err
		return res
	}
	res.Engine = engine
	if st, err := os.Stat(file); err == nil {
		res.Size = st.Size()
	}
	log.Printf("[%s] dump ok via %s (%s gzipped)", db.Name, engine, humanSize(res.Size))

	for i := range db.Destinations {
		d := &db.Destinations[i]
		log.Printf("[%s] delivering to %s", db.Name, d.Label())
		err := Deliver(ctx, d, file, db.Name, db.RetryCount())
		if err != nil && errors.Is(err, errDestUnavailable) && notify != nil {
			notify.Notify("Backup folder unavailable", fmt.Sprintf("%s: %v", db.Name, err))
		}
		res.Dests = append(res.Dests, DestResult{Label: d.Label(), Err: err})
	}
	return res
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
