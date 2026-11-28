//go:build !windows

package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

func testManager(t *testing.T) *Manager {
	t.Helper()
	t.Setenv("PMGO_HOME", t.TempDir())
	m := NewManager()
	t.Cleanup(m.StopAll)
	return m
}

func waitFor(t *testing.T, m *Manager, name string, pred func(common.ProcessInfo) bool) common.ProcessInfo {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last common.ProcessInfo
	for time.Now().Before(deadline) {
		infos, err := m.Describe(name)
		if err != nil {
			t.Fatal(err)
		}
		last = infos[0]
		if pred(last) {
			return last
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition not met for %s, last: status=%s restarts=%d unstable=%d", name, last.Status, last.Restarts, last.Unstable)
	return last
}

func shell(t *testing.T, name, script string) common.AppConfig {
	t.Helper()
	return common.AppConfig{Name: name, Script: "sh", Args: []string{"-c", script}, Cwd: t.TempDir()}
}

func TestCrashLoopEndsErrored(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "crash", "exit 3")
	cfg.MaxRestarts = 3
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	i := waitFor(t, m, "crash", func(i common.ProcessInfo) bool { return i.Status == common.StatusErrored })
	if i.Restarts != 2 || i.Unstable != 3 {
		t.Fatalf("restarts=%d unstable=%d, want 2 and 3", i.Restarts, i.Unstable)
	}
	if i.ExitCode == nil || *i.ExitCode != 3 {
		t.Fatalf("exit code = %v, want 3", i.ExitCode)
	}
}

func TestNoAutorestartStopsAfterCleanExit(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "once", "exit 0")
	off := false
	cfg.Autorestart = &off
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	i := waitFor(t, m, "once", func(i common.ProcessInfo) bool { return i.Status == common.StatusStopped })
	if i.Restarts != 0 {
		t.Fatalf("restarts = %d, want 0", i.Restarts)
	}
}

func TestStopExitCodeSkipsRestart(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "code", "exit 7")
	cfg.StopExitCodes = []int{7}
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "code", func(i common.ProcessInfo) bool { return i.Status == common.StatusStopped })
}

func TestStopKillsStubbornChild(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "stubborn", "trap '' TERM; while true; do sleep 0.1; done")
	cfg.KillTimeoutMs = 300
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	i := waitFor(t, m, "stubborn", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	start := time.Now()
	if _, err := m.Stop("stubborn"); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("stop took %v", d)
	}
	i = waitFor(t, m, "stubborn", func(i common.ProcessInfo) bool { return i.Status == common.StatusStopped })
	if i.PID != 0 {
		t.Fatalf("pid = %d after stop", i.PID)
	}
}

func TestRestartChangesPID(t *testing.T) {
	m := testManager(t)
	if _, err := m.Start(common.StartParams{Config: shell(t, "svc", "sleep 30")}); err != nil {
		t.Fatal(err)
	}
	a := waitFor(t, m, "svc", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	if _, err := m.Restart("svc"); err != nil {
		t.Fatal(err)
	}
	b := waitFor(t, m, "svc", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline && i.PID != a.PID })
	if b.PID == a.PID {
		t.Fatal("pid unchanged")
	}
}

func TestStartExistingNameUpdatesConfig(t *testing.T) {
	m := testManager(t)
	if _, err := m.Start(common.StartParams{Config: shell(t, "svc", "sleep 30")}); err != nil {
		t.Fatal(err)
	}
	cfg := shell(t, "svc", "sleep 31")
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	if got := len(m.List()); got != 1 {
		t.Fatalf("%d processes, want 1", got)
	}
	i := waitFor(t, m, "svc", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	if i.Config.Args[1] != "sleep 31" {
		t.Fatalf("config not updated: %v", i.Config.Args)
	}
}

func TestDeleteRemovesProcess(t *testing.T) {
	m := testManager(t)
	if _, err := m.Start(common.StartParams{Config: shell(t, "svc", "sleep 30")}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Delete("svc"); err != nil {
		t.Fatal(err)
	}
	if len(m.List()) != 0 {
		t.Fatal("process still listed")
	}
	if _, err := m.Describe("svc"); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestSaveAndResurrect(t *testing.T) {
	m := testManager(t)
	if _, err := m.Start(common.StartParams{Config: shell(t, "svc", "sleep 30")}); err != nil {
		t.Fatal(err)
	}
	if n, err := m.Save(); err != nil || n != 1 {
		t.Fatalf("save = %d, %v", n, err)
	}
	m.StopAll()
	m2 := NewManager()
	t.Cleanup(m2.StopAll)
	infos, err := m2.Resurrect()
	if err != nil || len(infos) != 1 {
		t.Fatalf("resurrect = %v, %v", infos, err)
	}
	waitFor(t, m2, "svc", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
}

func TestLogsAndEnvInjection(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "logger", `echo "name=$PMGO_NAME foo=$FOO"; echo oops >&2; sleep 30`)
	cfg.Env = map[string]string{"FOO": "bar"}
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	i := waitFor(t, m, "logger", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	deadline := time.Now().Add(3 * time.Second)
	for {
		b, _ := os.ReadFile(i.OutLog)
		if string(b) == "name=logger foo=bar\n" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("out log = %q", b)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if b, _ := os.ReadFile(i.ErrLog); string(b) != "oops\n" {
		t.Fatalf("err log = %q", b)
	}
	if err := m.Flush("logger"); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(i.OutLog); st.Size() != 0 {
		t.Fatal("out log not flushed")
	}
}

func TestStartFailureRetriesThenErrors(t *testing.T) {
	m := testManager(t)
	cfg := common.AppConfig{Name: "missing", Script: filepath.Join(t.TempDir(), "nope"), MaxRestarts: 2}
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "missing", func(i common.ProcessInfo) bool { return i.Status == common.StatusErrored })
}

func TestNamespaceTarget(t *testing.T) {
	m := testManager(t)
	for _, n := range []string{"a", "b"} {
		cfg := shell(t, n, "sleep 30")
		cfg.Namespace = "grp"
		if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
			t.Fatal(err)
		}
	}
	infos, err := m.Stop("grp")
	if err != nil || len(infos) != 2 {
		t.Fatalf("stop grp = %d, %v", len(infos), err)
	}
}
