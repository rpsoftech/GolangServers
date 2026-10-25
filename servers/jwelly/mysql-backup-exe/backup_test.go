package main

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

type recordingNotifier struct {
	mu   sync.Mutex
	msgs []string
}

func (r *recordingNotifier) Notify(title, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, title+": "+message)
}

func (r *recordingNotifier) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.msgs)
}

// bkFakeMysqldump writes a stand-in mysqldump that prints body and exits with
// code, and returns its path.
func bkFakeMysqldump(t *testing.T, body string, code int) string {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		p := filepath.Join(dir, "fake-mysqldump.cmd")
		script := "@echo off\r\n"
		for _, line := range strings.Split(body, "\n") {
			script += "@echo " + line + "\r\n"
		}
		script += "@exit /b " + string(rune('0'+code)) + "\r\n"
		if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	p := filepath.Join(dir, "fake-mysqldump.sh")
	script := "#!/bin/sh\ncat <<'EOT'\n" + body + "\nEOT\nexit " + string(rune('0'+code)) + "\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func bkDB(t *testing.T, dumpPath string, dests ...Destination) *DatabaseConfig {
	zero := 0
	return &DatabaseConfig{
		Name:          "shop",
		Engine:        engineMysqldump,
		MysqldumpPath: dumpPath,
		Retries:       &zero,
		MysqlConfig:   MysqlConfig{Host: "127.0.0.1", Port: 1, Username: "u", Database: "shop"},
		Destinations:  dests,
	}
}

const bkGoodDump = "-- fake dump\n-- Dump completed on 2026-01-01 00:00:00"

func TestRunBackupDeliversToLocalFolder(t *testing.T) {
	out := filepath.Join(t.TempDir(), "nested", "backups")
	db := bkDB(t, bkFakeMysqldump(t, bkGoodDump, 0), Destination{Type: destLocal, Path: out})
	n := &recordingNotifier{}

	res := RunBackup(context.Background(), db, n)
	if !res.OK() {
		t.Fatalf("backup failed: dump=%v dests=%+v", res.Err, res.Dests)
	}
	if res.Engine != engineMysqldump || res.Size == 0 {
		t.Errorf("engine=%q size=%d", res.Engine, res.Size)
	}
	files, _ := filepath.Glob(filepath.Join(out, "shop-*.sql.gz"))
	if len(files) != 1 {
		t.Fatalf("want 1 backup file in %s, got %v", out, files)
	}
	f, _ := os.Open(files[0])
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("backup is not gzip: %v", err)
	}
	body, _ := io.ReadAll(gz)
	if !strings.Contains(string(body), "Dump completed") {
		t.Errorf("unexpected dump content: %q", body)
	}
	if n.count() != 0 {
		t.Errorf("no notification expected, got %v", n.msgs)
	}
}

func TestRunBackupNotifiesWhenLocalFolderUnusable(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(t.TempDir(), "good")
	db := bkDB(t, bkFakeMysqldump(t, bkGoodDump, 0),
		Destination{Type: destLocal, Path: filepath.Join(blocker, "sub")},
		Destination{Type: destLocal, Path: good},
	)
	n := &recordingNotifier{}

	res := RunBackup(context.Background(), db, n)
	if res.OK() {
		t.Fatal("result must not be OK when a destination failed")
	}
	if len(res.Dests) != 2 || !errors.Is(res.Dests[0].Err, errDestUnavailable) || res.Dests[1].Err != nil {
		t.Fatalf("dests = %+v", res.Dests)
	}
	if n.count() != 1 || !strings.Contains(n.msgs[0], "Backup folder unavailable") {
		t.Errorf("notifications = %v", n.msgs)
	}
	if files, _ := filepath.Glob(filepath.Join(good, "*.sql.gz")); len(files) != 1 {
		t.Errorf("the working destination must still get the backup, got %v", files)
	}
}

func TestRunBackupDumpFailure(t *testing.T) {
	db := bkDB(t, bkFakeMysqldump(t, "partial", 2), Destination{Type: destLocal, Path: t.TempDir()})
	res := RunBackup(context.Background(), db, &recordingNotifier{})
	if res.OK() || res.Err == nil {
		t.Fatalf("want dump failure, got %+v", res)
	}
	if len(res.Dests) != 0 {
		t.Errorf("nothing must be delivered after a failed dump: %+v", res.Dests)
	}
}

func TestRunBackupLeavesNoTempFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	t.Setenv("TMPDIR", tmp)
	db := bkDB(t, bkFakeMysqldump(t, bkGoodDump, 0), Destination{Type: destLocal, Path: t.TempDir()})
	RunBackup(context.Background(), db, nil)
	entries, _ := os.ReadDir(tmp)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "mysql-backup-") {
			t.Errorf("temp dir %s was not cleaned up", e.Name())
		}
	}
}

func TestResultOK(t *testing.T) {
	if !(&Result{}).OK() {
		t.Error("empty result is OK")
	}
	if (&Result{Err: errors.New("x")}).OK() {
		t.Error("dump error is not OK")
	}
	if (&Result{Dests: []DestResult{{Err: errors.New("x")}}}).OK() {
		t.Error("destination error is not OK")
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{0: "0 B", 1023: "1023 B", 1024: "1.0 KB", 1536: "1.5 KB", 5 << 20: "5.0 MB", 3 << 30: "3.0 GB"}
	for in, want := range cases {
		if got := humanSize(in); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestReportDaemonResult(t *testing.T) {
	n := &recordingNotifier{}
	reportDaemonResult(&Result{Name: "a"}, n)
	if n.count() != 0 {
		t.Error("a successful backup must not notify")
	}

	reportDaemonResult(&Result{Name: "b", Err: errors.New("boom\nsecond line")}, n)
	if n.count() != 1 || !strings.Contains(n.msgs[0], "boom") || strings.Contains(n.msgs[0], "second line") {
		t.Errorf("dump failure notification = %v", n.msgs)
	}

	// An unusable folder was already announced by RunBackup, so it must not be
	// announced a second time.
	n = &recordingNotifier{}
	reportDaemonResult(&Result{Name: "c", Dests: []DestResult{{Label: "local x", Err: errDestUnavailable}}}, n)
	if n.count() != 0 {
		t.Errorf("unavailable folder must not notify twice, got %v", n.msgs)
	}

	reportDaemonResult(&Result{Name: "d", Dests: []DestResult{{Label: "fileServer u", Err: errors.New("500")}}}, n)
	if n.count() != 1 {
		t.Errorf("upload failure must notify, got %v", n.msgs)
	}
}

func TestRunQuickExitCodes(t *testing.T) {
	good := bkDB(t, bkFakeMysqldump(t, bkGoodDump, 0), Destination{Type: destLocal, Path: t.TempDir()})
	good.Name = "good"
	if code := runQuick(&Config{Databases: []*DatabaseConfig{good}}, &recordingNotifier{}); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	bad := bkDB(t, bkFakeMysqldump(t, "x", 1), Destination{Type: destLocal, Path: t.TempDir()})
	bad.Name = "bad"
	if code := runQuick(&Config{Databases: []*DatabaseConfig{good, bad}}, &recordingNotifier{}); code != 1 {
		t.Errorf("exit code = %d, want 1 when any backup fails", code)
	}
}
