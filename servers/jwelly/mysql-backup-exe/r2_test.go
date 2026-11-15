package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	r2TestAccessKey = "AKIDTEST"
	r2TestBucket    = "backups"
)

// fakeS3 is just enough of the S3 API for the R2 destination: put, list (v2)
// and delete, path-style, in one bucket.
type fakeS3 struct {
	srv *httptest.Server

	mu       sync.Mutex
	objects  map[string][]byte // key inside the bucket
	failPuts int               // answer this many PUTs with 403
	puts     int
	badAuth  int
}

func newFakeS3(t *testing.T) *fakeS3 {
	t.Helper()
	f := &fakeS3{objects: map[string][]byte{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeS3) dest(folder string) *Destination {
	return &Destination{
		Type: destR2, Endpoint: f.srv.URL, Bucket: r2TestBucket, FolderPath: folder,
		AccessKeyID: r2TestAccessKey, SecretAccessKey: "secret",
	}
}

func (f *fakeS3) keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ks []string
	for k := range f.objects {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func (f *fakeS3) put(key string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = data
}

func s3Error(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	fmt.Fprintf(w, "<Error><Code>%s</Code><Message>%s</Message></Error>", code, code)
}

func (f *fakeS3) handle(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Authorization"), "Credential="+r2TestAccessKey+"/") {
		f.mu.Lock()
		f.badAuth++
		f.mu.Unlock()
		s3Error(w, http.StatusForbidden, "InvalidAccessKeyId")
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/")
	bucket, key, _ := strings.Cut(rest, "/")
	if bucket != r2TestBucket {
		s3Error(w, http.StatusNotFound, "NoSuchBucket")
		return
	}
	switch r.Method {
	case http.MethodPut:
		f.mu.Lock()
		f.puts++
		if f.failPuts > 0 {
			f.failPuts--
			f.mu.Unlock()
			s3Error(w, http.StatusForbidden, "AccessDenied")
			return
		}
		f.mu.Unlock()
		var data []byte
		var err error
		if strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING-") {
			data, err = decodeAWSChunked(r.Body)
		} else {
			data, err = io.ReadAll(r.Body)
		}
		if err != nil {
			s3Error(w, http.StatusBadRequest, "BadRequest")
			return
		}
		f.put(key, data)
		w.Header().Set("ETag", `"fake"`)
	case http.MethodGet:
		f.list(w, r)
	case http.MethodDelete:
		f.mu.Lock()
		delete(f.objects, key)
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		s3Error(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
	}
}

func (f *fakeS3) list(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	delimiter := r.URL.Query().Get("delimiter")
	var contents, prefixes []string
	seen := map[string]bool{}
	for _, k := range f.keys() {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		if delimiter != "" {
			if i := strings.Index(k[len(prefix):], delimiter); i >= 0 {
				p := k[:len(prefix)+i+1]
				if !seen[p] {
					seen[p] = true
					prefixes = append(prefixes, p)
				}
				continue
			}
		}
		contents = append(contents, k)
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult><Name>` + r2TestBucket + `</Name>`)
	fmt.Fprintf(&b, "<Prefix>%s</Prefix><KeyCount>%d</KeyCount><IsTruncated>false</IsTruncated>", prefix, len(contents)+len(prefixes))
	for _, k := range contents {
		fmt.Fprintf(&b, "<Contents><Key>%s</Key><Size>1</Size><LastModified>2026-01-01T00:00:00.000Z</LastModified><ETag>&quot;x&quot;</ETag></Contents>", k)
	}
	for _, p := range prefixes {
		fmt.Fprintf(&b, "<CommonPrefixes><Prefix>%s</Prefix></CommonPrefixes>", p)
	}
	b.WriteString("</ListBucketResult>")
	w.Header().Set("Content-Type", "application/xml")
	io.WriteString(w, b.String())
}

// decodeAWSChunked reads an aws-chunked body, which S3 clients use over plain
// HTTP: "<hex size>;chunk-signature=...\r\n<data>\r\n" repeated, ending in a
// zero-size chunk.
func decodeAWSChunked(r io.Reader) ([]byte, error) {
	br := bufio.NewReader(r)
	var out bytes.Buffer
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		sizeText, _, _ := strings.Cut(strings.TrimSpace(line), ";")
		n, err := strconv.ParseInt(sizeText, 16, 64)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return out.Bytes(), nil
		}
		if _, err := io.CopyN(&out, br, n); err != nil {
			return nil, err
		}
		if _, err := br.Discard(2); err != nil {
			return nil, err
		}
	}
}

func TestUploadR2StoresUnderFolder(t *testing.T) {
	s3 := newFakeS3(t)
	dir := t.TempDir()
	data := writeRandomFile(t, dir, "shop-20260101-000001.sql.gz", 300_000)

	if err := uploadR2(context.Background(), s3.dest("/nightly/shop/"), filepath.Join(dir, "shop-20260101-000001.sql.gz")); err != nil {
		t.Fatal(err)
	}
	keys := s3.keys()
	if len(keys) != 1 || keys[0] != "nightly/shop/shop-20260101-000001.sql.gz" {
		t.Fatalf("keys = %v", keys)
	}
	if !bytes.Equal(s3.objects[keys[0]], data) {
		t.Error("stored object differs from the file")
	}
}

func TestUploadR2NoFolderUsesBucketRoot(t *testing.T) {
	s3 := newFakeS3(t)
	dir := t.TempDir()
	writeRandomFile(t, dir, "a.sql.gz", 1000)
	if err := uploadR2(context.Background(), s3.dest(""), filepath.Join(dir, "a.sql.gz")); err != nil {
		t.Fatal(err)
	}
	if keys := s3.keys(); len(keys) != 1 || keys[0] != "a.sql.gz" {
		t.Fatalf("keys = %v", keys)
	}
}

func TestUploadR2WrongCredentials(t *testing.T) {
	s3 := newFakeS3(t)
	dir := t.TempDir()
	writeRandomFile(t, dir, "a.sql.gz", 1000)
	d := s3.dest("")
	d.AccessKeyID = "someone-else"
	err := uploadR2(context.Background(), d, filepath.Join(dir, "a.sql.gz"))
	if err == nil || !strings.Contains(err.Error(), "r2 bucket "+r2TestBucket) {
		t.Fatalf("err = %v", err)
	}
	if len(s3.keys()) != 0 {
		t.Error("object stored despite bad credentials")
	}
}

func TestUploadR2MissingSource(t *testing.T) {
	s3 := newFakeS3(t)
	if err := uploadR2(context.Background(), s3.dest(""), filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error")
	}
}

func TestDeliverR2Retries(t *testing.T) {
	noBackoff(t)
	s3 := newFakeS3(t)
	s3.failPuts = 2
	dir := t.TempDir()
	writeRandomFile(t, dir, "shop-20260101-000001.sql.gz", 5000)

	err := Deliver(context.Background(), s3.dest("f"), filepath.Join(dir, "shop-20260101-000001.sql.gz"), "shop", 3)
	if err != nil {
		t.Fatal(err)
	}
	if s3.puts != 3 {
		t.Errorf("PUT attempts = %d, want 3", s3.puts)
	}
	if len(s3.keys()) != 1 {
		t.Errorf("keys = %v", s3.keys())
	}
}

func TestDeliverR2GivesUpAfterRetries(t *testing.T) {
	noBackoff(t)
	s3 := newFakeS3(t)
	s3.failPuts = 100
	dir := t.TempDir()
	writeRandomFile(t, dir, "shop-20260101-000001.sql.gz", 5000)

	err := Deliver(context.Background(), s3.dest(""), filepath.Join(dir, "shop-20260101-000001.sql.gz"), "shop", 2)
	if err == nil || !strings.Contains(err.Error(), "after 3 attempt(s)") {
		t.Fatalf("err = %v", err)
	}
	if s3.puts != 3 {
		t.Errorf("PUT attempts = %d, want 3", s3.puts)
	}
}

func TestDeliverR2PrunesOnlyThisDatabase(t *testing.T) {
	s3 := newFakeS3(t)
	for _, k := range []string{
		"f/shop-20260101-000001.sql.gz",
		"f/shop-20260102-000001.sql.gz",
		"f/shop-20260103-000001.sql.gz.age",
		"f/shop-eu-20260101-000001.sql.gz",  // another database
		"f/sub/shop-20260101-000001.sql.gz", // a sub-folder is not ours
		"other/shop-20260101-000001.sql.gz", // another prefix
		"f/notes.txt",
	} {
		s3.put(k, []byte("x"))
	}
	dir := t.TempDir()
	writeRandomFile(t, dir, "shop-20260301-000001.sql.gz", 100)
	d := s3.dest("f")
	d.KeepLast = 2

	if err := Deliver(context.Background(), d, filepath.Join(dir, "shop-20260301-000001.sql.gz"), "shop", 0); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"f/notes.txt",
		"f/shop-20260103-000001.sql.gz.age",
		"f/shop-20260301-000001.sql.gz",
		"f/shop-eu-20260101-000001.sql.gz",
		"f/sub/shop-20260101-000001.sql.gz",
		"other/shop-20260101-000001.sql.gz",
	}
	if got := s3.keys(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("keys after prune:\n got %v\nwant %v", got, want)
	}
}

func TestDeliverR2NoPruneByDefault(t *testing.T) {
	s3 := newFakeS3(t)
	s3.put("shop-20260101-000001.sql.gz", []byte("x"))
	dir := t.TempDir()
	writeRandomFile(t, dir, "shop-20260301-000001.sql.gz", 100)
	if err := Deliver(context.Background(), s3.dest(""), filepath.Join(dir, "shop-20260301-000001.sql.gz"), "shop", 0); err != nil {
		t.Fatal(err)
	}
	if len(s3.keys()) != 2 {
		t.Errorf("keys = %v, want both kept", s3.keys())
	}
}

func TestUploadR2StalledServer(t *testing.T) {
	shortStallTimeout(t, 300*time.Millisecond)
	// Accepts connections and never answers.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan net.Conn, 16)
	go func() {
		defer close(accepted)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted <- c
		}
	}()
	defer func() {
		ln.Close()
		for c := range accepted {
			c.Close()
		}
	}()

	file := filepath.Join(t.TempDir(), "big.sql.gz")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(file, 64<<20); err != nil {
		t.Fatal(err)
	}
	d := &Destination{Type: destR2, Endpoint: "http://" + ln.Addr().String(), Bucket: r2TestBucket, AccessKeyID: "k", SecretAccessKey: "s"}

	start := time.Now()
	err = uploadR2(context.Background(), d, file)
	if !errors.Is(err, errUploadStalled) {
		t.Fatalf("err = %v, want errUploadStalled", err)
	}
	if took := time.Since(start); took > 30*time.Second {
		t.Errorf("stall detected only after %s", took)
	}
}

func TestRunBackupEncryptedToR2(t *testing.T) {
	key, pub := newKey(t)
	s3 := newFakeS3(t)
	db := bkDB(t, bkFakeMysqldump(t, bkGoodDump, 0), *s3.dest("shop"))
	db.EncryptTo = []string{pub}

	res := RunBackup(context.Background(), db, &recordingNotifier{})
	if !res.OK() || !res.Encrypted {
		t.Fatalf("ok=%v encrypted=%v dump=%v dests=%+v", res.OK(), res.Encrypted, res.Err, res.Dests)
	}
	keys := s3.keys()
	if len(keys) != 1 || !strings.HasPrefix(keys[0], "shop/shop-") || !strings.HasSuffix(keys[0], ".sql.gz.age") {
		t.Fatalf("keys = %v", keys)
	}
	if bytes.Contains(s3.objects[keys[0]], []byte("Dump completed")) {
		t.Error("object holds plaintext")
	}
	enc := writeFile(t, t.TempDir(), "x.sql.gz.age", s3.objects[keys[0]])
	if err := decryptFile(enc, enc+".out", key); err != nil {
		t.Fatalf("uploaded object cannot be decrypted: %v", err)
	}
}

func TestR2ConfigValidation(t *testing.T) {
	good := Destination{Type: destR2, AccountID: "abc123", Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s"}
	cases := []struct {
		name   string
		change func(d *Destination)
		want   string // empty = valid
	}{
		{"valid with accountId", func(d *Destination) {}, ""},
		{"valid with endpoint", func(d *Destination) { d.AccountID = ""; d.Endpoint = "https://s3.example.com" }, ""},
		{"no account or endpoint", func(d *Destination) { d.AccountID = "" }, "accountId (or endpoint)"},
		{"no bucket", func(d *Destination) { d.Bucket = "" }, "bucket, accessKeyId and secretAccessKey"},
		{"no secret", func(d *Destination) { d.SecretAccessKey = "" }, "bucket, accessKeyId and secretAccessKey"},
		{"placeholder", func(d *Destination) { d.AccessKeyID = "<your-key>" }, "placeholder"},
		{"endpoint without scheme", func(d *Destination) { d.Endpoint = "s3.example.com" }, "full http(s) URL"},
		{"negative keepLast", func(d *Destination) { d.KeepLast = -1 }, "keepLast"},
		{"negative keepDays", func(d *Destination) { d.KeepDays = -1 }, "keepDays"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := good
			tc.change(&d)
			problems := strings.Join(d.validateR2(0), "; ")
			if tc.want == "" && problems != "" {
				t.Errorf("unexpected problems: %s", problems)
			}
			if tc.want != "" && !strings.Contains(problems, tc.want) {
				t.Errorf("problems = %q, want it to contain %q", problems, tc.want)
			}
		})
	}
}

func TestR2LabelAndPrefix(t *testing.T) {
	d := &Destination{Type: destR2, Bucket: "b", FolderPath: "/x/y/"}
	if got := d.Label(); got != "r2 b/x/y" {
		t.Errorf("label = %q", got)
	}
	if got := r2Prefix(d); got != "x/y/" {
		t.Errorf("prefix = %q", got)
	}
	if got := r2Prefix(&Destination{}); got != "" {
		t.Errorf("empty prefix = %q", got)
	}
}
