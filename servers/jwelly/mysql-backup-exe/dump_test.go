package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTailWriter(t *testing.T) {
	var sink bytes.Buffer
	tw := &tailWriter{w: &sink}

	small := []byte("hello")
	if n, err := tw.Write(small); n != 5 || err != nil {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if string(tw.tail) != "hello" {
		t.Fatalf("tail = %q", tw.tail)
	}

	big := bytes.Repeat([]byte("a"), 1000)
	big = append(big, []byte("END")...)
	if n, err := tw.Write(big); n != len(big) || err != nil {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if len(tw.tail) != 512 {
		t.Fatalf("tail len = %d, want 512", len(tw.tail))
	}
	if !strings.HasSuffix(string(tw.tail), "aaaEND") {
		t.Fatalf("tail does not end with the last bytes: %q", tw.tail[len(tw.tail)-10:])
	}
	if sink.Len() != 5+len(big) || sink.String() != "hello"+string(big) {
		t.Fatalf("not all bytes forwarded: %d", sink.Len())
	}

	// Many small writes still keep the last 512 bytes only.
	tw2 := &tailWriter{w: io.Discard}
	for i := 0; i < 1000; i++ {
		tw2.Write([]byte("0123456789"))
	}
	if len(tw2.tail) != 512 {
		t.Fatalf("tail len = %d, want 512", len(tw2.tail))
	}
}

func TestFirstLine(t *testing.T) {
	cases := map[string]string{
		"":                  "",
		"one":               "one",
		"one\ntwo":          "one",
		"\n  padded \nx":    "padded",
		"  trailing  \n\n ": "trailing",
	}
	for in, want := range cases {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// fakeEngine is a dumpEngine whose behaviour is set by the test.
type fakeEngine struct {
	name string
	body string
	err  error
}

func (f fakeEngine) Name() string { return f.name }
func (f fakeEngine) Dump(ctx context.Context, db *DatabaseConfig, w io.Writer) error {
	if _, err := io.WriteString(w, f.body); err != nil {
		return err
	}
	return f.err
}

func gunzipFile(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("not a gzip file: %v", err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gunzip: %v", err)
	}
	return string(b)
}

func TestDumpWith(t *testing.T) {
	db := &DatabaseConfig{Name: "x"}
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "ok.sql.gz")
		body := "CREATE TABLE t (a int);\n-- Dump completed on 2024-01-01 00:00:00\n"
		if err := dumpWith(ctx, fakeEngine{name: "fake", body: body}, db, p); err != nil {
			t.Fatal(err)
		}
		if got := gunzipFile(t, p); got != body {
			t.Fatalf("content = %q", got)
		}
	})

	t.Run("marker beyond tail window is incomplete", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "x.sql.gz")
		body := "-- Dump completed on x\n" + strings.Repeat("z", 600)
		err := dumpWith(ctx, fakeEngine{name: "fake", body: body}, db, p)
		if err == nil || !strings.Contains(err.Error(), "incomplete") {
			t.Fatalf("err = %v, want incomplete", err)
		}
	})

	t.Run("no marker", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "x.sql.gz")
		err := dumpWith(ctx, fakeEngine{name: "fake", body: "CREATE TABLE t (a int);\n"}, db, p)
		if err == nil || !strings.Contains(err.Error(), "incomplete") {
			t.Fatalf("err = %v, want incomplete", err)
		}
	})

	t.Run("engine error", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "x.sql.gz")
		boom := errors.New("boom")
		err := dumpWith(ctx, fakeEngine{name: "fake", body: "-- Dump completed on x\n", err: boom}, db, p)
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
	})

	t.Run("cannot create file", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "missing-dir", "x.sql.gz")
		if err := dumpWith(ctx, fakeEngine{name: "fake"}, db, p); err == nil {
			t.Fatal("expected error")
		}
	})
}

// writeFakeMysqldump creates a script that stands in for mysqldump. The script
// body is given for both platforms.
func writeFakeMysqldump(t *testing.T, dir, unixBody, winBody string) string {
	t.Helper()
	var p, content string
	var mode os.FileMode
	if runtime.GOOS == "windows" {
		p = filepath.Join(dir, "fakedump.cmd")
		content = "@echo off\r\n" + winBody
		mode = 0o644
	} else {
		p = filepath.Join(dir, "fakedump.sh")
		content = "#!/bin/sh\n" + unixBody
		mode = 0o755
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func testDB(engine, script string) *DatabaseConfig {
	return &DatabaseConfig{
		Name:          "mydb",
		Engine:        engine,
		MysqldumpPath: script,
		MysqlConfig: MysqlConfig{
			Host: "127.0.0.1", Port: 1, Username: "u",
			Password: "s3cr3t-pw", Database: "d",
		},
	}
}

func TestDumpToFileMysqldumpSuccess(t *testing.T) {
	scriptDir := t.TempDir()
	script := writeFakeMysqldump(t, scriptDir,
		`echo "CREATE TABLE t (a int);"
echo "-- pwd=$MYSQL_PWD"
echo "-- args=$*"
echo "-- Dump completed on x"
`,
		"echo CREATE TABLE t (a int);\r\necho -- pwd=%MYSQL_PWD%\r\necho -- args=%*\r\necho -- Dump completed on x\r\n")

	for _, engine := range []string{engineAuto, engineMysqldump} {
		out := t.TempDir()
		db := testDB(engine, script)
		path, used, err := DumpToFile(context.Background(), db, out)
		if err != nil {
			t.Fatalf("engine %s: %v", engine, err)
		}
		if used != engineMysqldump {
			t.Fatalf("engine %s: used = %q", engine, used)
		}
		if filepath.Dir(path) != out || !strings.HasPrefix(filepath.Base(path), "mydb-") || !strings.HasSuffix(path, ".sql.gz") {
			t.Fatalf("unexpected path %q", path)
		}
		got := gunzipFile(t, path)
		if !strings.Contains(got, "CREATE TABLE t (a int);") {
			t.Errorf("dump content missing: %q", got)
		}
		if !strings.Contains(got, "-- pwd=s3cr3t-pw") {
			t.Errorf("MYSQL_PWD not passed to mysqldump: %q", got)
		}
		args := got[strings.Index(got, "-- args="):]
		args = firstLine(args)
		if strings.Contains(args, "s3cr3t-pw") {
			t.Errorf("password leaked on the command line: %q", args)
		}
		for _, want := range []string{"--host=127.0.0.1", "--port=1", "--user=u", "--single-transaction", "--routines", "--triggers", "--events"} {
			if !strings.Contains(args, want) {
				t.Errorf("args %q missing %s", args, want)
			}
		}
		if !strings.HasSuffix(strings.TrimSpace(args), "d") {
			t.Errorf("database should be the last argument: %q", args)
		}
		if names := listDir(t, out); len(names) != 1 {
			t.Errorf("expected only the dump in out dir, got %v", names)
		}
	}
}

func TestDumpToFileFallbackAndFailures(t *testing.T) {
	scriptDir := t.TempDir()
	failing := writeFakeMysqldump(t, scriptDir,
		"echo \"mysqldump: bad things\" >&2\nexit 2\n",
		"echo mysqldump: bad things 1>&2\r\nexit /b 2\r\n")
	// Exits 0 but never writes the completion marker.
	incomplete := writeFakeMysqldump(t, t.TempDir(),
		"echo \"CREATE TABLE t (a int);\"\n",
		"echo CREATE TABLE t (a int);\r\n")

	t.Run("auto falls back to native", func(t *testing.T) {
		out := t.TempDir()
		path, used, err := DumpToFile(context.Background(), testDB(engineAuto, failing), out)
		if err == nil {
			t.Fatalf("expected error, got path %q engine %q", path, used)
		}
		if path != "" || used != "" {
			t.Errorf("path/engine should be empty on failure: %q %q", path, used)
		}
		msg := err.Error()
		if !strings.Contains(msg, engineMysqldump) || !strings.Contains(msg, engineNative) {
			t.Errorf("error should mention both engines: %v", err)
		}
		if !strings.Contains(msg, "bad things") {
			t.Errorf("mysqldump stderr missing from error: %v", err)
		}
		if names := listDir(t, out); len(names) != 0 {
			t.Errorf("leftover files after failure: %v", names)
		}
	})

	t.Run("mysqldump only", func(t *testing.T) {
		out := t.TempDir()
		_, _, err := DumpToFile(context.Background(), testDB(engineMysqldump, failing), out)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), engineMysqldump) {
			t.Errorf("error should mention mysqldump: %v", err)
		}
		if strings.Contains(err.Error(), engineNative+":") {
			t.Errorf("native engine must not run: %v", err)
		}
		if names := listDir(t, out); len(names) != 0 {
			t.Errorf("leftover files after failure: %v", names)
		}
	})

	t.Run("incomplete dump is rejected", func(t *testing.T) {
		out := t.TempDir()
		_, _, err := DumpToFile(context.Background(), testDB(engineMysqldump, incomplete), out)
		if err == nil || !strings.Contains(err.Error(), "incomplete") {
			t.Fatalf("err = %v, want incomplete", err)
		}
		if names := listDir(t, out); len(names) != 0 {
			t.Errorf("leftover files after failure: %v", names)
		}
	})

	t.Run("missing mysqldumpPath", func(t *testing.T) {
		out := t.TempDir()
		missing := filepath.Join(t.TempDir(), "nope", "mysqldump")
		_, _, err := DumpToFile(context.Background(), testDB(engineMysqldump, missing), out)
		if err == nil || !strings.Contains(err.Error(), "mysqldumpPath") {
			t.Fatalf("err = %v", err)
		}
		if names := listDir(t, out); len(names) != 0 {
			t.Errorf("leftover files after failure: %v", names)
		}
	})

	t.Run("native only unreachable", func(t *testing.T) {
		out := t.TempDir()
		_, _, err := DumpToFile(context.Background(), testDB(engineNative, ""), out)
		if err == nil || !strings.Contains(err.Error(), engineNative) {
			t.Fatalf("err = %v", err)
		}
		if names := listDir(t, out); len(names) != 0 {
			t.Errorf("leftover files after failure: %v", names)
		}
	})

	t.Run("cancelled context stops fallback", func(t *testing.T) {
		out := t.TempDir()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _, err := DumpToFile(ctx, testDB(engineAuto, failing), out)
		if err == nil {
			t.Fatal("expected error")
		}
		if strings.Contains(err.Error(), engineNative+":") {
			t.Errorf("native engine should not run after cancellation: %v", err)
		}
	})
}
