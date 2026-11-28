//go:build !windows

package daemon

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// lineSink collects log hook lines.
type lineSink struct {
	mu    sync.Mutex
	lines []string
}

func (s *lineSink) add(l string) { s.mu.Lock(); s.lines = append(s.lines, l); s.mu.Unlock() }
func (s *lineSink) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.lines...)
}

func eventually(t *testing.T, d time.Duration, what string, f func() bool) {
	t.Helper()
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if f() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out: %s", what)
}

func gone(pid int) bool { return !processAlive(pid) }

func TestLogTailRotatesByCopyTruncate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.log")
	sink := &lineSink{}
	tl, err := startTail(path, 1000, 3, sink.add)
	if err != nil {
		t.Fatal(err)
	}
	defer tl.Close()
	// The writer plays the child: one O_APPEND descriptor that stays open.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	want := 0
	for batch := 0; batch < 12; batch++ {
		for i := 0; i < 20; i++ {
			fmt.Fprintf(f, "line %04d %s\n", want, strings.Repeat("x", 20))
			want++
		}
		tl.sync()
	}
	got := sink.all()
	if len(got) != want {
		t.Fatalf("%d lines reported, want %d", len(got), want)
	}
	for i, l := range got {
		if !strings.HasPrefix(l, fmt.Sprintf("line %04d ", i)) {
			t.Fatalf("line %d = %q", i, l)
		}
	}
	eventually(t, 3*time.Second, "background compression finished", func() bool {
		pending, _ := filepath.Glob(path + ".*.rot")
		tmp, _ := filepath.Glob(path + ".*.tmp")
		_, err := os.Stat(path + ".1.gz")
		return len(pending) == 0 && len(tmp) == 0 && err == nil
	})
	st, _ := os.Stat(path)
	if st.Size() > 1000+20*31 {
		t.Fatalf("current file is %d bytes, rotation did not truncate", st.Size())
	}
	if _, err := os.Stat(path + ".4.gz"); err == nil {
		t.Fatal("kept more than retain files")
	}
	zf, err := os.Open(path + ".1.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer zf.Close()
	zr, err := gzip.NewReader(zf)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "line ") || !strings.HasSuffix(string(b), "\n") {
		t.Fatalf("rotated content looks wrong: %.40q", b)
	}
	// The child's descriptor still works and lines keep flowing after rotation.
	fmt.Fprintln(f, "after rotation")
	tl.sync()
	if got := sink.all(); got[len(got)-1] != "after rotation" {
		t.Fatalf("last line %q", got[len(got)-1])
	}
}

func TestLogTailTruncateAndPartialLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.log")
	sink := &lineSink{}
	tl, err := startTail(path, 0, 3, sink.add)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	defer f.Close()
	f.WriteString("one\ntw")
	tl.sync()
	f.WriteString("o\r\n")
	tl.sync()
	if err := tl.Truncate(); err != nil {
		t.Fatal(err)
	}
	f.WriteString("three\n")
	tl.sync()
	f.WriteString("tail")
	tl.Close() // reports the trailing partial line
	want := []string{"one", "two", "three", "tail"}
	if got := sink.all(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("lines = %q, want %q", got, want)
	}
}

func Test10kLinesNoLossNoDuplicate(t *testing.T) {
	m := testManager(t)
	sink := &lineSink{}
	m.SetHooks(nil, func(l common.LogLine) {
		if l.Stream == "out" {
			sink.add(l.Line)
		}
	})
	if _, err := m.Start(common.StartParams{Config: shell(t, "many", "seq 1 10000; sleep 30")}); err != nil {
		t.Fatal(err)
	}
	eventually(t, 15*time.Second, "10000 lines", func() bool { return len(sink.all()) >= 10000 })
	time.Sleep(600 * time.Millisecond) // a duplicate would show up now
	got := sink.all()
	if len(got) != 10000 {
		t.Fatalf("%d lines, want 10000", len(got))
	}
	for i, l := range got {
		if l != fmt.Sprint(i+1) {
			t.Fatalf("line %d = %q", i, l)
		}
	}
}

func TestChildKeepsWritingWithoutDaemonPipe(t *testing.T) {
	// The child's stdout is the log file itself, not a pipe of the daemon.
	m := testManager(t)
	if _, err := m.Start(common.StartParams{Config: shell(t, "fd", "ls -l /dev/fd/1 > $0.fd 2>&1; sleep 30")}); err != nil {
		t.Fatal(err)
	}
	i := waitFor(t, m, "fd", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	_ = i
	fi, err := os.Lstat(filepath.Join(common.LogsDir(), "fd-out.log"))
	if err != nil || fi.Mode()&os.ModeNamedPipe != 0 {
		t.Fatalf("log is not a regular file: %v %v", fi, err)
	}
}

func TestStateFileAtomicAndPrivate(t *testing.T) {
	m := testManager(t)
	m.EnableState(t.Context())
	if _, err := m.Start(common.StartParams{Config: shell(t, "st", "sleep 30")}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "st", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	var sf stateFile
	eventually(t, 3*time.Second, "state file with pid", func() bool {
		b, err := os.ReadFile(common.StateFile())
		if err != nil || json.Unmarshal(b, &sf) != nil {
			return false
		}
		return len(sf.Procs) == 1 && sf.Procs[0].PID > 0
	})
	r := sf.Procs[0]
	if r.StartTime == "" || r.Cmdline == "" || r.Status != common.StatusOnline || sf.Version != stateVersion {
		t.Fatalf("incomplete record: %+v", r)
	}
	if st, _ := os.Stat(common.StateFile()); st.Mode().Perm() != 0o600 {
		t.Fatalf("state mode %v", st.Mode())
	}
	left, _ := filepath.Glob(common.StateFile() + ".tmp*")
	if len(left) != 0 {
		t.Fatalf("temp files left: %v", left)
	}
}

func TestCorruptStateStartsFresh(t *testing.T) {
	m := testManager(t)
	if err := os.WriteFile(common.StateFile(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n := m.Adopt(); n != 0 || len(m.List()) != 0 {
		t.Fatalf("adopted %d from a corrupt file", n)
	}
	if _, err := os.Stat(common.StateFile() + ".corrupt"); err != nil {
		t.Fatalf("corrupt file not kept: %v", err)
	}
	if n := m.Adopt(); n != 0 { // missing file
		t.Fatal("adopted from a missing file")
	}
}

// orphan starts a process the way a dead daemon would have left it: own
// process group, stdout and stderr on the log files, nobody supervising.
func orphan(t *testing.T, name, script string) (procRecord, *exec.Cmd) {
	t.Helper()
	cfg := shell(t, name, script)
	p := newProc(nil, 0, 0, cfg, os.Environ())
	if err := os.MkdirAll(common.LogsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := os.OpenFile(p.outPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	errf, _ := os.OpenFile(p.errPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	cmd := exec.Command(cfg.Script, cfg.Args...)
	cmd.Stdout, cmd.Stderr = out, errf
	setProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	out.Close()
	errf.Close()
	go cmd.Wait() // reaps it like init would
	t.Cleanup(func() { signalGroup(cmd.Process.Pid, true) })
	start, _ := procStartTime(cmd.Process.Pid)
	line, _ := procCmdline(cmd.Process.Pid)
	return procRecord{ID: 5, Instance: 0, Config: cfg, Env: os.Environ(), PID: cmd.Process.Pid,
		StartTime: start, Cmdline: line, Restarts: 2, Unstable: 1, CreatedAt: time.Now().UnixMilli(),
		StartedAt: time.Now().Add(-time.Minute).UnixMilli(), Status: common.StatusOnline}, cmd
}

func writeState(t *testing.T, next int, recs ...procRecord) {
	t.Helper()
	b, _ := json.Marshal(stateFile{Version: stateVersion, NextID: next, Procs: recs})
	if err := os.WriteFile(common.StateFile(), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAdoptKeepsPidCountersAndStreamsLogs(t *testing.T) {
	m := testManager(t)
	rec, _ := orphan(t, "web", `while true; do echo tick; sleep 0.1; done`)
	writeState(t, 9, rec)
	sink := &lineSink{}
	var evMu sync.Mutex
	var events []common.Event
	m.SetHooks(func(e common.Event) { evMu.Lock(); events = append(events, e); evMu.Unlock() },
		func(l common.LogLine) { sink.add(l.Line) })
	if n := m.Adopt(); n != 1 {
		t.Fatalf("adopted %d, want 1", n)
	}
	i := waitFor(t, m, "web", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	if i.PID != rec.PID || i.ID != 5 || i.Restarts != 2 || i.Unstable != 1 {
		t.Fatalf("info = pid %d id %d restarts %d unstable %d", i.PID, i.ID, i.Restarts, i.Unstable)
	}
	if i.UptimeMs < 50_000 {
		t.Fatalf("uptime %dms, want the original start time", i.UptimeMs)
	}
	eventually(t, 5*time.Second, "log lines reach the hook", func() bool { return len(sink.all()) >= 3 })
	if m.nextID != 9 {
		t.Fatalf("nextID = %d, want 9", m.nextID)
	}
	evMu.Lock()
	adopted := len(events) > 0 && events[0].Type == "online" && events[0].Reason == "adopt"
	evMu.Unlock()
	if !adopted {
		t.Fatalf("events = %+v", events)
	}
	// Stop and delete still work on an adopted process.
	if _, err := m.Stop("web"); err != nil {
		t.Fatal(err)
	}
	eventually(t, 5*time.Second, "adopted process stopped", func() bool { return gone(rec.PID) })
	if i := waitFor(t, m, "web", func(i common.ProcessInfo) bool { return i.Status == common.StatusStopped }); i.PID != 0 {
		t.Fatalf("pid after stop = %d", i.PID)
	}
}

func TestDeleteAndKillStopAdoptedProcesses(t *testing.T) {
	m := testManager(t)
	a, _ := orphan(t, "a", "sleep 60")
	b, _ := orphan(t, "b", "sleep 61")
	a.ID, b.ID = 1, 2
	writeState(t, 3, a, b)
	if n := m.Adopt(); n != 2 {
		t.Fatalf("adopted %d", n)
	}
	if _, err := m.Delete("a"); err != nil {
		t.Fatal(err)
	}
	eventually(t, 5*time.Second, "delete stops a", func() bool { return gone(a.PID) })
	if !processAlive(b.PID) {
		t.Fatal("delete of a stopped b")
	}
	m.StopAll() // what `pm2go kill` does
	eventually(t, 5*time.Second, "kill stops b", func() bool { return gone(b.PID) })
}

func TestAdoptedAppThatDiedWhileDaemonDownIsRestarted(t *testing.T) {
	m := testManager(t)
	rec, cmd := orphan(t, "gone", "sleep 60")
	_ = cmd.Process.Kill()
	eventually(t, 5*time.Second, "orphan dead", func() bool { return gone(rec.PID) })
	writeState(t, 6, rec)
	if n := m.Adopt(); n != 0 {
		t.Fatalf("adopted %d, want 0", n)
	}
	i := waitFor(t, m, "gone", func(i common.ProcessInfo) bool {
		return i.Status == common.StatusOnline && i.PID != 0 && i.PID != rec.PID
	})
	if i.Restarts != rec.Restarts+1 {
		t.Fatalf("restarts = %d, want %d", i.Restarts, rec.Restarts+1)
	}
}

func TestAdoptedAppThatDiedWithoutAutorestartIsErrored(t *testing.T) {
	m := testManager(t)
	rec, cmd := orphan(t, "noauto", "sleep 60")
	off := false
	rec.Config.Autorestart = &off
	_ = cmd.Process.Kill()
	eventually(t, 5*time.Second, "orphan dead", func() bool { return gone(rec.PID) })
	writeState(t, 6, rec)
	m.Adopt()
	i := waitFor(t, m, "noauto", func(i common.ProcessInfo) bool { return i.Status == common.StatusErrored })
	if i.PID != 0 {
		t.Fatalf("pid = %d", i.PID)
	}
}

func TestStoppedEntriesStayStopped(t *testing.T) {
	m := testManager(t)
	rec := procRecord{ID: 1, Config: shell(t, "off", "sleep 60"), Status: common.StatusStopped}
	writeState(t, 2, rec)
	m.Adopt()
	i := waitFor(t, m, "off", func(i common.ProcessInfo) bool { return true })
	time.Sleep(200 * time.Millisecond)
	if i.Status != common.StatusStopped || i.PID != 0 {
		t.Fatalf("info = %+v", i)
	}
}

func TestPIDReuseIsNeverAdoptedOrKilled(t *testing.T) {
	// An unrelated program that happens to own the recorded pid.
	other := exec.Command("sleep", "60")
	setProcAttr(other)
	if err := other.Start(); err != nil {
		t.Fatal(err)
	}
	go other.Wait()
	t.Cleanup(func() { signalGroup(other.Process.Pid, true) })
	realStart, _ := procStartTime(other.Process.Pid)
	realCmd, _ := procCmdline(other.Process.Pid)

	for name, tc := range map[string]struct{ start, cmdline string }{
		"other start time": {"1", realCmd},
		"other cmdline":    {realStart, "sh -c something else"},
	} {
		t.Run(name, func(t *testing.T) {
			m := testManager(t)
			rec := procRecord{ID: 1, Config: shell(t, "victim", "sleep 61"), PID: other.Process.Pid,
				StartTime: tc.start, Cmdline: tc.cmdline, Status: common.StatusOnline, Restarts: 4}
			writeState(t, 2, rec)
			if n := m.Adopt(); n != 0 {
				t.Fatalf("adopted %d, want 0", n)
			}
			i := waitFor(t, m, "victim", func(i common.ProcessInfo) bool {
				return i.Status == common.StatusOnline && i.PID != 0
			})
			if i.PID == other.Process.Pid {
				t.Fatal("supervising the unrelated pid")
			}
			if i.Restarts != 5 {
				t.Fatalf("restarts = %d, want 5", i.Restarts)
			}
			m.StopAll()
			if _, err := m.Delete("victim"); err != nil {
				t.Fatal(err)
			}
			time.Sleep(200 * time.Millisecond)
			if !processAlive(other.Process.Pid) {
				t.Fatal("the unrelated process was killed")
			}
		})
	}
}

// --- real daemon processes -------------------------------------------------

type testDaemon struct {
	home   string
	cmd    *exec.Cmd
	exited chan struct{}
}

func startDaemon(t *testing.T, home string) *testDaemon {
	t.Helper()
	logf, err := os.OpenFile(filepath.Join(home, "daemon.out"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "PMGO_HOME="+home, "PMGO_TEST_DAEMON=1")
	cmd.Stdout, cmd.Stderr = logf, logf
	setProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	logf.Close()
	d := &testDaemon{home: home, cmd: cmd, exited: make(chan struct{})}
	go func() { cmd.Wait(); close(d.exited) }()
	t.Cleanup(func() { _ = cmd.Process.Kill(); <-d.exited })
	eventually(t, 10*time.Second, "daemon socket", func() bool {
		var out map[string]int
		return d.call("ping", nil, &out) == nil
	})
	return d
}

func (d *testDaemon) call(method string, params, out any) error {
	c, err := net.DialTimeout("unix", filepath.Join(d.home, "rpc.sock"), time.Second)
	if err != nil {
		return err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(20 * time.Second))
	req := common.Request{Method: method}
	if params != nil {
		req.Params, _ = json.Marshal(params)
	}
	if err := json.NewEncoder(c).Encode(req); err != nil {
		return err
	}
	line, err := bufio.NewReader(c).ReadBytes('\n')
	if err != nil {
		return err
	}
	var resp common.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	if out != nil {
		return json.Unmarshal(resp.Data, out)
	}
	return nil
}

func (d *testDaemon) list(t *testing.T) map[string]common.ProcessInfo {
	t.Helper()
	var infos []common.ProcessInfo
	if err := d.call(common.MethodList, nil, &infos); err != nil {
		t.Fatal(err)
	}
	out := map[string]common.ProcessInfo{}
	for _, i := range infos {
		out[i.Name] = i
	}
	return out
}

func (d *testDaemon) waitOnline(t *testing.T, names ...string) map[string]common.ProcessInfo {
	t.Helper()
	var cur map[string]common.ProcessInfo
	defer func() {
		if t.Failed() {
			t.Log(d.dump(t))
		}
	}()
	eventually(t, 10*time.Second, "apps online", func() bool {
		cur = d.list(t)
		for _, n := range names {
			if cur[n].Status != common.StatusOnline || cur[n].PID == 0 {
				return false
			}
		}
		return true
	})
	return cur
}

// waitStateHasPIDs waits until the state file records the given pids; the file
// is written right after a spawn, so a test that kills the daemon at once
// would otherwise race the writer.
func waitStateHasPIDs(t *testing.T, infos map[string]common.ProcessInfo) {
	t.Helper()
	eventually(t, 5*time.Second, "state file records the pids", func() bool {
		b, _ := os.ReadFile(common.StateFile())
		for _, i := range infos {
			if !strings.Contains(string(b), fmt.Sprintf(`"pid": %d,`, i.PID)) {
				return false
			}
		}
		return true
	})
}

func (d *testDaemon) dump(t *testing.T) string {
	b, _ := os.ReadFile(filepath.Join(d.home, "daemon.out"))
	return fmt.Sprintf("%v\n%s", d.list(t), b)
}

func (d *testDaemon) sigkill(t *testing.T) {
	t.Helper()
	_ = d.cmd.Process.Signal(syscall.SIGKILL)
	select {
	case <-d.exited:
	case <-time.After(5 * time.Second):
		t.Fatal("daemon did not die")
	}
}

func fileSize(p string) int64 {
	st, err := os.Stat(p)
	if err != nil {
		return -1
	}
	return st.Size()
}

func TestDaemonKilledAndRestartedKeepsProcesses(t *testing.T) {
	home, err := os.MkdirTemp("", "pg")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("PMGO_HOME", home)
	d1 := startDaemon(t, home)
	names := []string{"a", "b", "c"}
	for _, n := range names {
		cfg := shell(t, n, `while true; do echo $PMGO_NAME tick; sleep 0.1; done`)
		cfg.KillTimeoutMs = 2000
		if err := d1.call(common.MethodStart, common.StartParams{Config: cfg}, nil); err != nil {
			t.Fatal(err)
		}
	}
	before := d1.waitOnline(t, names...)
	t.Cleanup(func() { // never leave apps behind
		for _, i := range before {
			signalGroup(i.PID, true)
		}
		var cur []common.ProcessInfo
		if err := (&testDaemon{home: home}).call(common.MethodList, nil, &cur); err == nil {
			for _, i := range cur {
				signalGroup(i.PID, true)
			}
		}
	})

	waitStateHasPIDs(t, before)
	// 1. kill -9 the daemon: the apps keep running and keep logging.
	logA := before["a"].OutLog
	d1.sigkill(t)
	size := fileSize(logA)
	for _, i := range before {
		if !processAlive(i.PID) {
			t.Fatalf("%s died with the daemon", i.Name)
		}
	}
	eventually(t, 5*time.Second, "log grows while the daemon is down", func() bool { return fileSize(logA) > size })

	// 2. a new daemon adopts the same pids with the same counters.
	d2 := startDaemon(t, home)
	after := d2.waitOnline(t, names...)
	for _, n := range names {
		if after[n].PID != before[n].PID || after[n].Restarts != 0 || after[n].ID != before[n].ID {
			t.Fatalf("%s: before pid %d id %d, after pid %d id %d restarts %d", n,
				before[n].PID, before[n].ID, after[n].PID, after[n].ID, after[n].Restarts)
		}
	}

	// 3. an adopted app that crashes is restarted by the new daemon.
	_ = syscall.Kill(-after["b"].PID, syscall.SIGKILL)
	cur := map[string]common.ProcessInfo{}
	eventually(t, 10*time.Second, "b restarted by the adopting daemon", func() bool {
		cur = d2.list(t)
		return cur["b"].Status == common.StatusOnline && cur["b"].PID != after["b"].PID && cur["b"].PID != 0
	})
	if cur["b"].Restarts != 1 {
		t.Fatalf("b restarts = %d, want 1", cur["b"].Restarts)
	}

	// 4. an app that exits while the daemon is down is restarted per policy.
	waitStateHasPIDs(t, cur)
	d2.sigkill(t)
	_ = syscall.Kill(-cur["c"].PID, syscall.SIGKILL)
	eventually(t, 5*time.Second, "c dead", func() bool { return gone(cur["c"].PID) })
	d3 := startDaemon(t, home)
	fin := d3.waitOnline(t, names...)
	if fin["a"].PID != before["a"].PID || fin["b"].PID != cur["b"].PID {
		t.Fatalf("a or b changed pid: %+v", fin)
	}
	if fin["c"].PID == cur["c"].PID || fin["c"].Restarts != 1 {
		t.Fatalf("c pid %d restarts %d, want a new pid and 1 restart", fin["c"].PID, fin["c"].Restarts)
	}

	// 5. delete and kill still stop adopted processes.
	waitStateHasPIDs(t, fin)
	d3.sigkill(t)
	d4 := startDaemon(t, home)
	d4.waitOnline(t, names...)
	if err := d4.call(common.MethodDelete, common.TargetParams{Target: "a"}, nil); err != nil {
		t.Fatal(err)
	}
	eventually(t, 5*time.Second, "a stopped by delete", func() bool { return gone(fin["a"].PID) })
	if err := d4.call(common.MethodKill, nil, nil); err != nil {
		t.Fatal(err)
	}
	eventually(t, 10*time.Second, "kill stops b and c", func() bool {
		return gone(fin["b"].PID) && gone(fin["c"].PID)
	})
	eventually(t, 5*time.Second, "state file removed by kill", func() bool {
		_, err := os.Stat(common.StateFile())
		return os.IsNotExist(err)
	})
}

func TestSIGTERMLeavesApplicationsRunning(t *testing.T) {
	home, err := os.MkdirTemp("", "pg")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("PMGO_HOME", home)
	d := startDaemon(t, home)
	if err := d.call(common.MethodStart, common.StartParams{Config: shell(t, "keep", "sleep 60")}, nil); err != nil {
		t.Fatal(err)
	}
	pid := d.waitOnline(t, "keep")["keep"].PID
	t.Cleanup(func() { signalGroup(pid, true) })
	if err := d.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	eventually(t, 10*time.Second, "daemon exit", func() bool { return d.call("ping", nil, nil) != nil })
	time.Sleep(300 * time.Millisecond)
	if !processAlive(pid) {
		t.Fatal("SIGTERM to the daemon stopped the application")
	}
	b, err := os.ReadFile(common.StateFile())
	if err != nil || !strings.Contains(string(b), fmt.Sprintf(`"pid": %d`, pid)) {
		t.Fatalf("state after SIGTERM: %v %s", err, b)
	}
}

func TestConcurrentStartReloadScaleSaveList(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "busy", "sleep 30")
	cfg.Instances = 2
	cfg.KillTimeoutMs = 500
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	run := func(f func(i int)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
					f(i)
				}
			}
		}()
	}
	run(func(i int) {
		c := cfg
		c.Env = map[string]string{"N": fmt.Sprint(i)} // a config update restarts the app
		_, _ = m.Start(common.StartParams{Config: c})
	})
	run(func(int) { _, _ = m.Reload("busy") })
	run(func(i int) { _, _ = m.Scale("busy", 1+i%3) })
	run(func(int) { _, _ = m.Save() })
	run(func(int) { _ = m.List(); _, _ = m.Describe("busy") })
	run(func(int) { m.persistNow() })
	time.Sleep(1500 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestFiftyRapidRotationsLoseNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.log")
	sink := &lineSink{}
	tl, err := startTail(path, 100, 60, sink.add)
	if err != nil {
		t.Fatal(err)
	}
	defer tl.Close()
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	defer f.Close()
	for i := 0; i < 50; i++ {
		fmt.Fprintf(f, "rot %02d %s\n", i, strings.Repeat("y", 120))
		tl.sync() // each sync rotates: the file is over the limit
	}
	eventually(t, 5*time.Second, "all archives compressed", func() bool {
		pending, _ := filepath.Glob(path + ".*.rot")
		tmp, _ := filepath.Glob(path + ".*.tmp")
		_, err := os.Stat(path + ".50.gz")
		return len(pending) == 0 && len(tmp) == 0 && err == nil
	})
	// .1.gz is the newest, .50.gz the oldest: every line exactly once, in order.
	for n := 1; n <= 50; n++ {
		zf, err := os.Open(fmt.Sprintf("%s.%d.gz", path, n))
		if err != nil {
			t.Fatalf("archive %d missing: %v", n, err)
		}
		zr, err := gzip.NewReader(zf)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(zr)
		zf.Close()
		if want := fmt.Sprintf("rot %02d ", 50-n); !strings.HasPrefix(string(b), want) {
			t.Fatalf("archive %d starts %.12q, want %q", n, b, want)
		}
	}
	if _, err := os.Stat(path + ".51.gz"); err == nil {
		t.Fatal("unexpected 51st archive")
	}
	if got := sink.all(); len(got) != 50 {
		t.Fatalf("%d lines reported, want 50", len(got))
	}
}

func TestRotationFailureKeepsLogging(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.log")
	sink := &lineSink{}
	tl, err := startTail(path, 50, 3, sink.add)
	if err != nil {
		t.Fatal(err)
	}
	defer tl.Close()
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	defer f.Close()
	// Rotation cannot create its copy: the directory is read-only.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o700)
	fmt.Fprintln(f, strings.Repeat("z", 80))
	tl.sync()
	if _, err := fmt.Fprintln(f, "still writing"); err != nil {
		t.Fatalf("child write failed after a rotation error: %v", err)
	}
	tl.sync()
	got := sink.all()
	if len(got) != 2 || got[1] != "still writing" {
		t.Fatalf("lines = %q", got)
	}
}
