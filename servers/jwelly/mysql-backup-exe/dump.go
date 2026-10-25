package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const dumpCompletedMarker = "Dump completed"

// dumpEngine writes a plain-text SQL dump of one database to w.
type dumpEngine interface {
	Name() string
	Dump(ctx context.Context, db *DatabaseConfig, w io.Writer) error
}

// tailWriter forwards writes and remembers the last bytes, so a dump can be
// checked for its trailing "Dump completed" line before it is trusted.
type tailWriter struct {
	w    io.Writer
	tail []byte
}

func (t *tailWriter) Write(p []byte) (int, error) {
	t.tail = append(t.tail, p...)
	if len(t.tail) > 512 {
		t.tail = t.tail[len(t.tail)-512:]
	}
	return t.w.Write(p)
}

// DumpToFile dumps db into a new gzip file in dir and returns its path and the
// engine that produced it. With engine "auto" mysqldump is tried first and the
// native Go dumper takes over if it is missing or fails.
func DumpToFile(ctx context.Context, db *DatabaseConfig, dir string) (path, engineUsed string, err error) {
	var engines []dumpEngine
	switch db.Engine {
	case engineMysqldump:
		engines = []dumpEngine{mysqldumpEngine{}}
	case engineNative:
		engines = []dumpEngine{nativeEngine{}}
	default:
		engines = []dumpEngine{mysqldumpEngine{}, nativeEngine{}}
	}

	name := fmt.Sprintf("%s-%s.sql.gz", db.Name, time.Now().Format("20060102-150405"))
	path = filepath.Join(dir, name)

	var errs []error
	for _, e := range engines {
		err := dumpWith(ctx, e, db, path)
		if err == nil {
			return path, e.Name(), nil
		}
		os.Remove(path)
		errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
		if len(engines) > 1 && e != engines[len(engines)-1] {
			log.Printf("[%s] %s engine failed (%s), falling back", db.Name, e.Name(), firstLine(err.Error()))
		}
		if ctx.Err() != nil {
			break
		}
	}
	return "", "", errors.Join(errs...)
}

func dumpWith(ctx context.Context, e dumpEngine, db *DatabaseConfig, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tail := &tailWriter{w: gz}
	dumpErr := e.Dump(ctx, db, tail)
	if err := gz.Close(); dumpErr == nil {
		dumpErr = err
	}
	if dumpErr == nil && !bytes.Contains(tail.tail, []byte(dumpCompletedMarker)) {
		dumpErr = errors.New("dump is incomplete (missing trailing \"Dump completed\" line)")
	}
	if err := f.Close(); dumpErr == nil && err != nil {
		dumpErr = err
	}
	return dumpErr
}

// --- mysqldump binary ---

type mysqldumpEngine struct{}

func (mysqldumpEngine) Name() string { return engineMysqldump }

func (mysqldumpEngine) Dump(ctx context.Context, db *DatabaseConfig, w io.Writer) error {
	bin, err := findMysqldump(db.MysqldumpPath)
	if err != nil {
		return err
	}
	m := db.MysqlConfig
	cmd := exec.CommandContext(ctx, bin,
		"--protocol=tcp",
		"--host="+m.Host,
		"--port="+strconv.Itoa(m.Port),
		"--user="+m.Username,
		"--single-transaction",
		"--routines", "--triggers", "--events",
		"--add-drop-table",
		"--default-character-set=utf8mb4",
		m.Database,
	)
	// The password goes through the environment, not the command line, so it
	// does not show up in the process list.
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+m.Password)
	var stderr bytes.Buffer
	cmd.Stdout = w
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func findMysqldump(configured string) (string, error) {
	if configured != "" {
		if _, err := os.Stat(configured); err != nil {
			return "", fmt.Errorf("mysqldumpPath %q: %w", configured, err)
		}
		return configured, nil
	}
	if p, err := exec.LookPath("mysqldump"); err == nil {
		return p, nil
	}
	// Next to the executable, for bundled installs.
	if exe, err := os.Executable(); err == nil {
		for _, n := range []string{"mysqldump.exe", "mysqldump"} {
			p := filepath.Join(filepath.Dir(exe), n)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}
	return "", errors.New("mysqldump not found on PATH")
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(line)
}
