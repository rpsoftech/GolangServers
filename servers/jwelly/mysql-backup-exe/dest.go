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
	"sort"
	"strings"
	"time"
)

// errDestUnavailable marks a destination that cannot be used right now (e.g.
// a missing or unreachable folder). The user is notified about these.
var errDestUnavailable = errors.New("destination unavailable")

const localFolderTimeout = 15 * time.Second

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
	if d.KeepLast > 0 {
		if err := pruneLocal(d.Path, dbName, d.KeepLast); err != nil {
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

// pruneLocal keeps the newest keep backups of dbName. File names carry a
// sortable timestamp, so name order is age order.
func pruneLocal(dir, dbName string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	prefix := dbName + "-"
	var names []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasPrefix(n, prefix) && strings.HasSuffix(n, ".sql.gz") {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	var errs []error
	for len(names) > keep {
		if err := os.Remove(filepath.Join(dir, names[0])); err != nil {
			errs = append(errs, err)
		}
		names = names[1:]
	}
	return errors.Join(errs...)
}

// --- file server ---

// uploadFile POSTs the backup as multipart form data. The body length is
// computed up front so the request is not chunked.
func uploadFile(ctx context.Context, d *Destination, file string) error {
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
	body := io.MultiReader(&head, f, strings.NewReader(tail))
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
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("server returned %d: %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}
