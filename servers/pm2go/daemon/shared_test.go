//go:build !windows

package daemon

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

var (
	appOnce sync.Once
	appPath string
	appErr  error
)

// sharedApp compiles testdata/sharedapp once per test run.
func sharedApp(t *testing.T) string {
	t.Helper()
	appOnce.Do(func() {
		dir, err := os.MkdirTemp("", "pmgo-app")
		if err != nil {
			appErr = err
			return
		}
		appPath = filepath.Join(dir, "sharedapp")
		if out, err := exec.Command("go", "build", "-o", appPath, "./testdata/sharedapp").CombinedOutput(); err != nil {
			appErr = fmt.Errorf("build helper: %v\n%s", err, out)
		}
	})
	if appErr != nil {
		t.Fatal(appErr)
	}
	return appPath
}

func TestMain(m *testing.M) {
	if os.Getenv("PMGO_TEST_DAEMON") != "" { // re-executed as a real daemon process
		if err := Run(Options{Resurrect: os.Getenv("PMGO_TEST_RESURRECT") != ""}); err != nil {
			fmt.Fprintln(os.Stderr, "daemon:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	code := m.Run()
	if appPath != "" {
		_ = os.RemoveAll(filepath.Dir(appPath))
	}
	os.Exit(code)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func sharedCfg(t *testing.T, name string, instances int) (common.AppConfig, string) {
	addr := freeAddr(t)
	return common.AppConfig{
		Name: name, Script: sharedApp(t), Cwd: t.TempDir(), Instances: instances,
		Listen: addr, WaitReady: true, KillTimeoutMs: 4000,
	}, addr
}

var noKeepAlive = &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}

func get(addr string) (string, error) {
	resp, err := noKeepAlive.Get("http://" + addr + "/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	return string(b), nil
}

func allOnline(t *testing.T, m *Manager, name string, n int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		ok := 0
		for _, i := range m.List() {
			if i.Name == name && i.Status == common.StatusOnline {
				ok++
			}
		}
		if ok == n {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s: %d instances not online", name, n)
}

func TestSharedSocketServedByManyPids(t *testing.T) {
	m := testManager(t)
	cfg, addr := sharedCfg(t, "web", 3)
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	allOnline(t, m, "web", 3)
	pids := map[string]int{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				pid, err := get(addr)
				if err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				pids[pid]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(pids) < 2 {
		t.Fatalf("requests served by %d pid(s), want more than 1: %v", len(pids), pids)
	}
}

func TestReloadDropsNoRequests(t *testing.T) {
	m := testManager(t)
	cfg, addr := sharedCfg(t, "web", 2)
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	allOnline(t, m, "web", 2)
	before := map[int]int{}
	for _, i := range m.List() {
		before[i.ID] = i.PID
	}

	var sent, failed atomic.Int64
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { // about 200 requests per second
		defer wg.Done()
		tick := time.NewTicker(5 * time.Millisecond)
		defer tick.Stop()
		var inflight sync.WaitGroup
		defer inflight.Wait()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				inflight.Add(1)
				go func() {
					defer inflight.Done()
					sent.Add(1)
					if _, err := get(addr); err != nil {
						failed.Add(1)
						t.Logf("request failed: %v", err)
					}
				}()
			}
		}
	}()
	time.Sleep(300 * time.Millisecond)
	infos, err := m.Reload("web")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()
	for _, i := range infos {
		if i.PID == before[i.ID] || i.Status != common.StatusOnline {
			t.Fatalf("instance %d not reloaded: %+v", i.ID, i)
		}
	}
	if sent.Load() < 100 {
		t.Fatalf("only %d requests sent", sent.Load())
	}
	if failed.Load() != 0 {
		t.Fatalf("%d of %d requests failed during reload", failed.Load(), sent.Load())
	}
}

func TestWaitReadyNeverSignalsIsKilled(t *testing.T) {
	m := testManager(t)
	cfg, _ := sharedCfg(t, "mute", 1)
	cfg.Env = map[string]string{"APP_NO_READY": "1"}
	cfg.ListenTimeoutMs = 400
	cfg.MaxRestarts = 2
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	i := waitFor(t, m, "mute", func(i common.ProcessInfo) bool { return i.Status == common.StatusErrored })
	if i.Unstable < 2 {
		t.Fatalf("unstable = %d, want at least 2", i.Unstable)
	}
	if i.PID != 0 {
		t.Fatalf("pid %d still set", i.PID)
	}
}

func TestWaitReadySlowAppIsWaitedFor(t *testing.T) {
	m := testManager(t)
	cfg, addr := sharedCfg(t, "slow", 1)
	cfg.Env = map[string]string{"APP_START_DELAY_MS": "900"}
	cfg.ListenTimeoutMs = 5000
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if i, _ := m.Describe("slow"); i[0].Status != common.StatusLaunching {
		t.Fatalf("status = %s while the app starts, want launching", i[0].Status)
	}
	i := waitFor(t, m, "slow", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	if i.Unstable != 0 || i.Restarts != 0 {
		t.Fatalf("unstable=%d restarts=%d, want 0", i.Unstable, i.Restarts)
	}
	if _, err := get(addr); err != nil {
		t.Fatal(err)
	}
}

func TestReloadFailureKeepsOldInstance(t *testing.T) {
	m := testManager(t)
	cfg, addr := sharedCfg(t, "keep", 1)
	cfg.ListenTimeoutMs = 600
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	allOnline(t, m, "keep", 1)
	old := m.List()[0].PID
	// A config whose new process never signals: change the env through Proc.
	m.mu.Lock()
	p := m.procs[0]
	m.mu.Unlock()
	p.mu.Lock()
	p.cfg.Env = map[string]string{"APP_NO_READY": "1"}
	p.mu.Unlock()
	if _, err := m.Reload("keep"); err == nil {
		t.Fatal("reload succeeded though the new instance never became ready")
	}
	if got := m.List()[0]; got.PID != old || got.Status != common.StatusOnline {
		t.Fatalf("old instance lost: %+v", got)
	}
	if _, err := get(addr); err != nil {
		t.Fatal(err)
	}
}

func fdCount(t *testing.T) int {
	t.Helper()
	ents, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Skip("no /dev/fd on this platform")
	}
	return len(ents)
}

// settledFDs returns the fd count once it stops falling (exec's copy
// goroutines and pipes from earlier tests close asynchronously).
func settledFDs(t *testing.T) int {
	t.Helper()
	last := fdCount(t)
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		n := fdCount(t)
		if n == last && i > 2 {
			return n
		}
		last = n
	}
	return last
}

func TestNoFDLeakOnDeleteAndScale(t *testing.T) {
	m := testManager(t)
	base := settledFDs(t)
	cfg, addr := sharedCfg(t, "leak", 2)
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	allOnline(t, m, "leak", 2)
	if _, err := m.Reload("leak"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Scale("leak", 4); err != nil {
		t.Fatal(err)
	}
	allOnline(t, m, "leak", 4)
	if _, err := m.Scale("leak", 0); err != nil {
		t.Fatal(err)
	}
	// The socket must be closed: nothing listens and the port can be bound again.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("port still held after scale to zero: %v", err)
	}
	l.Close()
	if n := settledFDs(t); n > base {
		t.Fatalf("fd count %d after scale to zero, want at most %d", n, base)
	}

	cfg2, _ := sharedCfg(t, "leak2", 2)
	if _, err := m.Start(common.StartParams{Config: cfg2}); err != nil {
		t.Fatal(err)
	}
	allOnline(t, m, "leak2", 2)
	if _, err := m.Delete("leak2"); err != nil {
		t.Fatal(err)
	}
	if n := settledFDs(t); n > base {
		t.Fatalf("fd count %d after delete, want at most %d", n, base)
	}
}

func TestReadReadyAcceptsCRLFAndSpaces(t *testing.T) {
	for _, in := range []string{"ready\n", "ready\r\n", "  ready  \n", "ready"} {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		ready := make(chan struct{})
		go readReady(r, ready)
		w.WriteString(in)
		w.Close()
		select {
		case <-ready:
		case <-time.After(2 * time.Second):
			t.Fatalf("input %q was not accepted as ready", in)
		}
		r.Close()
	}
	r, w, _ := os.Pipe()
	ready := make(chan struct{})
	go readReady(r, ready)
	w.WriteString("not ready\n")
	w.Close()
	select {
	case <-ready:
		t.Fatal(`"not ready" must not count as ready`)
	case <-time.After(200 * time.Millisecond):
	}
	r.Close()
}
