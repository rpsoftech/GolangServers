package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

func TestRotatingFileRotatesAndCallsLines(t *testing.T) {
	dir := t.TempDir()
	var mu sync.Mutex
	var lines []string
	r, err := openRotating(filepath.Join(dir, "a.log"), 100, 3, func(l string) {
		mu.Lock()
		lines = append(lines, l)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if _, err := r.Write([]byte(strings.Repeat("x", 30) + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	r.Write([]byte("partial"))
	r.Close()
	mu.Lock()
	defer mu.Unlock()
	if len(lines) != 21 || lines[20] != "partial" {
		t.Fatalf("lines = %d last=%q", len(lines), lines[len(lines)-1])
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "a.log.1.gz")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no rotated .1.gz file")
		}
		time.Sleep(20 * time.Millisecond)
	}
	st, _ := os.Stat(filepath.Join(dir, "a.log"))
	if st.Size() > 100 {
		t.Fatalf("current file %d bytes, want <= 100", st.Size())
	}
	// at most retain rotated files, none beyond
	if _, err := os.Stat(filepath.Join(dir, "a.log.4.gz")); err == nil {
		t.Fatal("kept more than retain files")
	}
}

func TestInstancesScaleAndNames(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "web", `echo id=$PMGO_INSTANCE_ID; sleep 30`)
	cfg.Instances = 3
	infos, err := m.Start(common.StartParams{Config: cfg})
	if err != nil || len(infos) != 3 {
		t.Fatalf("start = %d, %v", len(infos), err)
	}
	for _, i := range infos {
		if !strings.Contains(i.OutLog, "web-") {
			t.Fatalf("instance log path %q", i.OutLog)
		}
	}
	if infos, err = m.Scale("web", 1); err != nil || len(infos) != 1 {
		t.Fatalf("scale down = %d, %v", len(infos), err)
	}
	if infos, err = m.Scale("web", 4); err != nil || len(infos) != 4 {
		t.Fatalf("scale up = %d, %v", len(infos), err)
	}
	if got := len(m.List()); got != 4 {
		t.Fatalf("%d procs, want 4", got)
	}
}

func TestMaxMemoryRestartRequest(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "mem", "sleep 30")
	cfg.MaxMemoryRestartMB = 1
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	a := waitFor(t, m, "mem", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	ps, _ := m.resolve("mem")
	ps[0].mu.Lock()
	ps[0].use = usage{rss: 5 << 20}
	ps[0].mu.Unlock()
	// simulate the monitor decision without needing real RSS
	ps[0].requestRestart("max_memory")
	b := waitFor(t, m, "mem", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline && i.PID != a.PID })
	if b.Restarts != 1 || b.Unstable != 0 {
		t.Fatalf("restarts=%d unstable=%d, want 1 and 0", b.Restarts, b.Unstable)
	}
}

func TestWatchRestartsOnChange(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "w", "sleep 30")
	cfg.Watch = true
	cfg.WatchDelayMs = 50
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	a := waitFor(t, m, "w", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	time.Sleep(1200 * time.Millisecond) // let the first snapshot be taken
	if err := os.WriteFile(filepath.Join(cfg.Cwd, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "w", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline && i.PID != a.PID })
}

func TestEventsAndLogHook(t *testing.T) {
	m := testManager(t)
	var mu sync.Mutex
	var evs []common.Event
	var logs []common.LogLine
	m.SetHooks(func(e common.Event) { mu.Lock(); evs = append(evs, e); mu.Unlock() },
		func(l common.LogLine) { mu.Lock(); logs = append(logs, l); mu.Unlock() })
	cfg := shell(t, "ev", "echo hello; echo bad >&2; sleep 30")
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "ev", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		n := len(logs)
		mu.Unlock()
		if n >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	m.Stop("ev")
	mu.Lock()
	defer mu.Unlock()
	streams := map[string]string{}
	for _, l := range logs {
		streams[l.Stream] = l.Line
	}
	if streams["out"] != "hello" || streams["err"] != "bad" {
		t.Fatalf("logs = %v", logs)
	}
	var types []string
	for _, e := range evs {
		types = append(types, e.Type)
	}
	want := []string{"online", "exit", "stopped"}
	if strings.Join(types, ",") != strings.Join(want, ",") {
		t.Fatalf("events = %v, want %v", types, want)
	}
	if !evs[2].Manual {
		t.Fatal("stop event should be manual")
	}
}

func TestReloadKeepsOthersServing(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "r", "sleep 30")
	cfg.Instances = 2
	cfg.MinUptimeMs = 100
	if _, err := m.Start(common.StartParams{Config: cfg}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "r", func(i common.ProcessInfo) bool { return i.Status == common.StatusOnline })
	before := map[int]int{}
	for _, i := range m.List() {
		before[i.ID] = i.PID
	}
	infos, err := m.Reload("r")
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range infos {
		if i.PID == before[i.ID] || i.Status != common.StatusOnline {
			t.Fatalf("instance %d not reloaded: %+v", i.ID, i)
		}
	}
}

func TestInvalidCronRejected(t *testing.T) {
	m := testManager(t)
	cfg := shell(t, "c", "sleep 1")
	cfg.CronRestart = "not a cron"
	if _, err := m.Start(common.StartParams{Config: cfg}); err == nil {
		t.Fatal("expected error")
	}
}
