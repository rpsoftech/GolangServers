package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// errDestUnavailable marks a destination that cannot be used right now (e.g.
// a missing or unreachable folder). The user is notified about these.
var errDestUnavailable = errors.New("destination unavailable")

const localFolderTimeout = 15 * time.Second

// uploadStallTimeout aborts an upload attempt when no file data has been sent
// for this long. A slow but working link keeps going; a hung one is dropped
// and retried. A variable so tests can shorten it.
var uploadStallTimeout = 10 * time.Minute

var errUploadStalled = errors.New("upload stalled: no data sent")

// retryBackoff is the wait before retry number attempt (1-based). A variable
// so tests do not have to sleep.
var retryBackoff = func(attempt int) time.Duration {
	return time.Duration(attempt*attempt) * 5 * time.Second
}

// Deliver sends the finished backup to one destination, retrying uploads.
func Deliver(ctx context.Context, d *Destination, file, dbName string, retries int) error {
	switch d.Type {
	case destLocal:
		return deliverLocal(d, file, dbName)
	case destFileServer:
		var err error
		for attempt := 0; attempt <= retries; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryBackoff(attempt)):
				}
			}
			if err = uploadFile(ctx, d, file); err == nil {
				return nil
			}
		}
		return fmt.Errorf("after %d attempt(s): %w", retries+1, err)
	}
	return fmt.Errorf("unknown destination type %q", d.Type)
}

// --- local folder ---

func deliverLocal(d *Destination, file, dbName string) error {
	if err := ensureDir(d.Path); err != nil {
		return fmt.Errorf("%w: cannot use folder %s: %v", errDestUnavailable, d.Path, err)
	}
	target := filepath.Join(d.Path, filepath.Base(file))
	if err := copyFile(file, target); err != nil {
		return fmt.Errorf("copy to %s: %w", target, err)
	}
	if d.KeepLast > 0 || d.KeepDays > 0 {
		if err := pruneLocal(d.Path, dbName, d.KeepLast, d.KeepDays); err != nil {
			return fmt.Errorf("backup saved to %s but pruning old backups failed: %w", target, err)
		}
	}
	return nil
}

// ensureDir creates the folder if it is missing. It runs with a timeout
// because an unreachable network share can block for a long time.
func ensureDir(dir string) error {
	done := make(chan error, 1)
	go func() { done <- os.MkdirAll(dir, 0o755) }()
	select {
	case err := <-done:
		return err
	case <-time.After(localFolderTimeout):
		return fmt.Errorf("timed out after %s", localFolderTimeout)
	}
}

// copyFile writes to a .part file and renames it, so a half-copied backup is
// never mistaken for a finished one.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	part := dst + ".part"
	out, err := os.Create(part)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(part)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(part)
		return err
	}
	return os.Rename(part, dst)
}

// pruneLocal deletes dbName's backups beyond the newest keepLast or older than
// keepDays days; 0 disables a rule. The newest backup is always kept.
//
// Only names of exactly <dbName>-<timestamp>.sql.gz count, so databases
// sharing a folder never prune each other ("shop" must not match
// "shop-eu-..."). The timestamp in the name is the dump time and sorts
// chronologically, so it gives both the order and the age.
func pruneLocal(dir, dbName string, keepLast, keepDays int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(dbName) + `-(\d{8}-\d{6})\.sql\.gz$`)
	type backup struct {
		name  string
		taken time.Time
	}
	var backups []backup
	for _, e := range entries {
		m := pattern.FindStringSubmatch(e.Name())
		if e.IsDir() || m == nil {
			continue
		}
		taken, err := time.ParseInLocation(backupTimeLayout, m[1], time.Local)
		if err != nil {
			continue
		}
		backups = append(backups, backup{e.Name(), taken})
	}
	// Newest first.
	sort.Slice(backups, func(i, j int) bool { return backups[i].name > backups[j].name })

	cutoff := time.Now().AddDate(0, 0, -keepDays)
	var errs []error
	for i, b := range backups {
		if i == 0 {
			continue
		}
		tooMany := keepLast > 0 && i >= keepLast
		tooOld := keepDays > 0 && b.taken.Before(cutoff)
		if !tooMany && !tooOld {
			continue
		}
		if err := os.Remove(filepath.Join(dir, b.name)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// --- file server ---

// uploadFile POSTs the backup as multipart form data. The body length is
// computed up front so the request is not chunked.
func uploadFile(ctx context.Context, d *Destination, file string) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	name := filepath.Base(file)

	var head bytes.Buffer
	mw := multipart.NewWriter(&head)
	if err := mw.WriteField("path", d.FolderPath); err != nil {
		return err
	}
	if _, err := mw.CreateFormFile("file", name); err != nil {
		return err
	}
	tail := "\r\n--" + mw.Boundary() + "--\r\n"

	u, err := url.Parse(d.URL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, name)

	contentLength := int64(head.Len()) + st.Size() + int64(len(tail))
	// The transport reads the file as fast as the connection takes it, so
	// file reads track what is actually sent. Waiting for the response after
	// the file is sent is bounded by the job timeout instead: the server may
	// take a while to store a large backup.
	progress := &progressReader{r: f}
	progress.touch()
	go watchStall(ctx, cancel, progress)
	body := io.MultiReader(&head, progress, strings.NewReader(tail))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
	if err != nil {
		return err
	}
	req.ContentLength = contentLength
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Authorization", "Bearer "+d.Token)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		if cause := context.Cause(ctx); errors.Is(cause, errUploadStalled) {
			return fmt.Errorf("%w for %s", cause, uploadStallTimeout)
		}
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("server returned %d: %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// watchStall cancels ctx with errUploadStalled when r has not been read from
// for uploadStallTimeout. It stops once r is fully read or ctx ends.
func watchStall(ctx context.Context, cancel context.CancelCauseFunc, r *progressReader) {
	ticker := time.NewTicker(uploadStallTimeout / 10)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if r.eof.Load() {
				return
			}
			if r.idle() > uploadStallTimeout {
				cancel(errUploadStalled)
				return
			}
		}
	}
}

// progressReader records when data was last read from r and whether r is
// exhausted.
type progressReader struct {
	r        io.Reader
	lastRead atomic.Int64 // unix nanoseconds
	eof      atomic.Bool
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.touch()
	}
	if err == io.EOF {
		p.eof.Store(true)
	}
	return n, err
}

func (p *progressReader) touch() { p.lastRead.Store(time.Now().UnixNano()) }

func (p *progressReader) idle() time.Duration {
	return time.Since(time.Unix(0, p.lastRead.Load()))
}
