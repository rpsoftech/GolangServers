package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func noBackoff(t *testing.T) {
	t.Helper()
	old := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = old })
}

func writeRandomFile(t *testing.T, dir, name string, size int) []byte {
	t.Helper()
	data := make([]byte, size)
	rand.New(rand.NewSource(42)).Read(data)
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return data
}

type capturedUpload struct {
	method, path, auth, ctype string
	contentLength             int64
	transferEncoding          []string
	bytesRead                 int64
	folder                    string
	filename                  string
	fileData                  []byte
	fileField                 string
}

func TestUploadFileRequestShape(t *testing.T) {
	dir := t.TempDir()
	name := "db1-20240101-000000.sql.gz"
	data := writeRandomFile(t, dir, name, 300*1024+17)

	var got capturedUpload
	var parseErr error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method = r.Method
		got.path = r.URL.Path
		got.auth = r.Header.Get("Authorization")
		got.ctype = r.Header.Get("Content-Type")
		got.contentLength = r.ContentLength
		got.transferEncoding = r.TransferEncoding
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			parseErr = err
			return
		}
		got.bytesRead = int64(len(raw))
		_, params, err := mime.ParseMediaType(got.ctype)
		if err != nil {
			parseErr = err
			return
		}
		mr := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				parseErr = err
				return
			}
			b, _ := io.ReadAll(p)
			if p.FileName() != "" {
				got.fileField = p.FormName()
				got.filename = p.FileName()
				got.fileData = b
			} else if p.FormName() == "path" {
				got.folder = string(b)
			}
		}
	}))
	defer srv.Close()

	d := &Destination{Type: destFileServer, URL: srv.URL + "/api/upload", Token: "tok123", FolderPath: "backups/db1"}
	if err := uploadFile(context.Background(), d, filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	if parseErr != nil {
		t.Fatalf("server parse error: %v", parseErr)
	}
	if got.method != http.MethodPost {
		t.Errorf("method = %s", got.method)
	}
	if got.path != "/api/upload/"+name {
		t.Errorf("path = %s", got.path)
	}
	if got.auth != "Bearer tok123" {
		t.Errorf("auth = %q", got.auth)
	}
	mt, params, err := mime.ParseMediaType(got.ctype)
	if err != nil || mt != "multipart/form-data" || params["boundary"] == "" {
		t.Errorf("content-type = %q (%v)", got.ctype, err)
	}
	if len(got.transferEncoding) != 0 {
		t.Errorf("request is chunked: %v", got.transferEncoding)
	}
	if got.contentLength <= 0 || got.contentLength != got.bytesRead {
		t.Errorf("ContentLength %d != bytes read %d", got.contentLength, got.bytesRead)
	}
	if got.folder != "backups/db1" {
		t.Errorf("path field = %q", got.folder)
	}
	if got.fileField != "file" || got.filename != name {
		t.Errorf("file part = %q / %q", got.fileField, got.filename)
	}
	if !bytes.Equal(got.fileData, data) {
		t.Errorf("file content differs (got %d bytes, want %d)", len(got.fileData), len(data))
	}
}

func TestUploadFileNon2xx(t *testing.T) {
	dir := t.TempDir()
	writeRandomFile(t, dir, "a.sql.gz", 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		http.Error(w, "disk full", http.StatusInsufficientStorage)
	}))
	defer srv.Close()
	err := uploadFile(context.Background(), &Destination{URL: srv.URL}, filepath.Join(dir, "a.sql.gz"))
	if err == nil || !strings.Contains(err.Error(), "507") || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("err = %v", err)
	}
}

func TestUploadFileHugeErrorBodyTruncated(t *testing.T) {
	dir := t.TempDir()
	writeRandomFile(t, dir, "a.sql.gz", 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(strings.Repeat("x", 10000)))
	}))
	defer srv.Close()
	err := uploadFile(context.Background(), &Destination{URL: srv.URL}, filepath.Join(dir, "a.sql.gz"))
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v", err)
	}
	if n := strings.Count(err.Error(), "x"); n != 1024 {
		t.Errorf("body x count = %d, want 1024", n)
	}
}

func TestUploadFileMissingSource(t *testing.T) {
	err := uploadFile(context.Background(), &Destination{URL: "http://127.0.0.1:1"}, filepath.Join(t.TempDir(), "nope.sql.gz"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUploadFileBadURL(t *testing.T) {
	dir := t.TempDir()
	writeRandomFile(t, dir, "a.sql.gz", 10)
	err := uploadFile(context.Background(), &Destination{URL: "http://bad host/%zz"}, filepath.Join(dir, "a.sql.gz"))
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
}

func TestDeliverFileServerRetries(t *testing.T) {
	noBackoff(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "a.sql.gz")
	writeRandomFile(t, dir, "a.sql.gz", 1000)

	newSrv := func(failFirst int32, count *int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.Copy(io.Discard, r.Body)
			n := atomic.AddInt32(count, 1)
			if n <= failFirst {
				http.Error(w, "boom", 500)
			}
		}))
	}

	t.Run("succeeds after two failures", func(t *testing.T) {
		var c int32
		srv := newSrv(2, &c)
		defer srv.Close()
		err := Deliver(context.Background(), &Destination{Type: destFileServer, URL: srv.URL}, file, "a", 2)
		if err != nil {
			t.Fatal(err)
		}
		if c != 3 {
			t.Errorf("requests = %d, want 3", c)
		}
	})

	t.Run("always failing with retries=1", func(t *testing.T) {
		var c int32
		srv := newSrv(1<<30, &c)
		defer srv.Close()
		err := Deliver(context.Background(), &Destination{Type: destFileServer, URL: srv.URL}, file, "a", 1)
		if err == nil || !strings.Contains(err.Error(), "after 2 attempt(s)") || !strings.Contains(err.Error(), "500") {
			t.Fatalf("err = %v", err)
		}
		if c != 2 {
			t.Errorf("requests = %d, want 2", c)
		}
	})

	t.Run("retries=0", func(t *testing.T) {
		var c int32
		srv := newSrv(1<<30, &c)
		defer srv.Close()
		err := Deliver(context.Background(), &Destination{Type: destFileServer, URL: srv.URL}, file, "a", 0)
		if err == nil || !strings.Contains(err.Error(), "after 1 attempt(s)") {
			t.Fatalf("err = %v", err)
		}
		if c != 1 {
			t.Errorf("requests = %d, want 1", c)
		}
	})
}

func TestDeliverFileServerCancelDuringBackoff(t *testing.T) {
	old := retryBackoff
	retryBackoff = func(int) time.Duration { return time.Hour }
	t.Cleanup(func() { retryBackoff = old })

	dir := t.TempDir()
	file := filepath.Join(dir, "a.sql.gz")
	writeRandomFile(t, dir, "a.sql.gz", 100)

	var c int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		atomic.AddInt32(&c, 1)
		http.Error(w, "boom", 500)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Deliver(ctx, &Destination{Type: destFileServer, URL: srv.URL}, file, "a", 3)
	}()
	// wait for the first attempt to finish, then cancel while in backoff
	deadline := time.Now().Add(5 * time.Second)
	for atomic.LoadInt32(&c) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Deliver did not return promptly after cancel")
	}
	if n := atomic.LoadInt32(&c); n != 1 {
		t.Errorf("requests = %d, want 1", n)
	}
}

func TestDeliverLocalCopiesAndCreatesFolder(t *testing.T) {
	src := t.TempDir()
	name := "db-20240101-000000.sql.gz"
	data := writeRandomFile(t, src, name, 50*1024)
	target := filepath.Join(t.TempDir(), "a", "b", "c")

	d := &Destination{Type: destLocal, Path: target}
	if err := Deliver(context.Background(), d, filepath.Join(src, name), "db", 0); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(target, name))
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("copied content mismatch (err=%v)", err)
	}
	entries, _ := os.ReadDir(target)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".part") {
			t.Errorf("leftover part file %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("entries = %d, want 1", len(entries))
	}
}

func TestDeliverLocalUnavailable(t *testing.T) {
	src := t.TempDir()
	name := "db-20240101-000000.sql.gz"
	writeRandomFile(t, src, name, 10)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := &Destination{Type: destLocal, Path: filepath.Join(blocker, "sub")}
	err := Deliver(context.Background(), d, filepath.Join(src, name), "db", 0)
	if err == nil || !errors.Is(err, errDestUnavailable) {
		t.Fatalf("err = %v, want errDestUnavailable", err)
	}
}

func setupPruneDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range []string{
		"db-20240101.sql.gz", "db-20240102.sql.gz", "db-20240103.sql.gz", "db-20240104.sql.gz",
		"other-20200101.sql.gz", "db-20240101.txt", "db-notes.sql", "db-20240105.sql.gz.part",
	} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// a directory that looks like a backup must be left alone
	if err := os.Mkdir(filepath.Join(dir, "db-20240100.sql.gz"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func listNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

func TestPruneLocal(t *testing.T) {
	dir := setupPruneDir(t)
	if err := pruneLocal(dir, "db", 2); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"db-20240100.sql.gz", "db-20240101.txt", "db-20240103.sql.gz", "db-20240104.sql.gz",
		"db-20240105.sql.gz.part", "db-notes.sql", "other-20200101.sql.gz",
	}
	if got := listNames(t, dir); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestPruneLocalFewerThanKeep(t *testing.T) {
	dir := setupPruneDir(t)
	before := listNames(t, dir)
	if err := pruneLocal(dir, "db", 10); err != nil {
		t.Fatal(err)
	}
	if got := listNames(t, dir); !reflect.DeepEqual(got, before) {
		t.Errorf("files changed: %v", got)
	}
}

func TestPruneLocalMissingDir(t *testing.T) {
	if err := pruneLocal(filepath.Join(t.TempDir(), "nope"), "db", 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeliverLocalKeepLast(t *testing.T) {
	src := t.TempDir()
	newName := "db-20240110.sql.gz"
	writeRandomFile(t, src, newName, 10)

	t.Run("keep 2", func(t *testing.T) {
		dir := setupPruneDir(t)
		d := &Destination{Type: destLocal, Path: dir, KeepLast: 2}
		if err := Deliver(context.Background(), d, filepath.Join(src, newName), "db", 0); err != nil {
			t.Fatal(err)
		}
		want := []string{
			"db-20240100.sql.gz", "db-20240101.txt", "db-20240104.sql.gz", "db-20240105.sql.gz.part",
			"db-20240110.sql.gz", "db-notes.sql", "other-20200101.sql.gz",
		}
		if got := listNames(t, dir); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v\nwant %v", got, want)
		}
	})

	t.Run("keep 0 keeps everything", func(t *testing.T) {
		dir := setupPruneDir(t)
		before := listNames(t, dir)
		d := &Destination{Type: destLocal, Path: dir, KeepLast: 0}
		if err := Deliver(context.Background(), d, filepath.Join(src, newName), "db", 0); err != nil {
			t.Fatal(err)
		}
		want := append(append([]string{}, before...), newName)
		sort.Strings(want)
		if got := listNames(t, dir); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v\nwant %v", got, want)
		}
	})
}

func TestDeliverUnknownType(t *testing.T) {
	err := Deliver(context.Background(), &Destination{Type: "ftp"}, "x", "db", 0)
	if err == nil || !strings.Contains(err.Error(), "unknown destination type") {
		t.Fatalf("err = %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	data := writeRandomFile(t, dir, "src.bin", 10000)
	dst := filepath.Join(dir, "dst.bin")
	if err := copyFile(filepath.Join(dir, "src.bin"), dst); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if !bytes.Equal(got, data) {
		t.Error("content differs")
	}
	if _, err := os.Stat(dst + ".part"); !os.IsNotExist(err) {
		t.Errorf(".part left behind: %v", err)
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dst.bin")
	if err := copyFile(filepath.Join(dir, "missing"), dst); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(dst + ".part"); !os.IsNotExist(err) {
		t.Errorf(".part exists: %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("dst exists: %v", err)
	}
}

func TestDestLabelStrings(t *testing.T) {
	cases := []struct {
		d    Destination
		want string
	}{
		{Destination{Type: destFileServer, URL: "https://x.example/up"}, "fileServer https://x.example/up"},
		{Destination{Type: destLocal, Path: `D:\bk`}, `local D:\bk`},
		{Destination{Type: "weird"}, "weird"},
	}
	for _, c := range cases {
		if got := c.d.Label(); got != c.want {
			t.Errorf("Label() = %q, want %q", got, c.want)
		}
	}
}
