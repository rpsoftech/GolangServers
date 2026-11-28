package cli

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// shortHome returns a short PMGO_HOME under /tmp (unix socket path limit).
func shortHome(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "pgcli")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	t.Setenv("PMGO_HOME", d)
	return d
}

// capture runs f and returns what it wrote to stdout and stderr.
func capture(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()
	oo, oe := os.Stdout, os.Stderr
	ro, wo, _ := os.Pipe()
	re, we, _ := os.Pipe()
	os.Stdout, os.Stderr = wo, we
	var wg sync.WaitGroup
	var so, se []byte
	wg.Add(2)
	go func() { defer wg.Done(); so, _ = io.ReadAll(ro) }()
	go func() { defer wg.Done(); se, _ = io.ReadAll(re) }()
	func() {
		defer func() { os.Stdout, os.Stderr = oo, oe }()
		f()
	}()
	wo.Close()
	we.Close()
	wg.Wait()
	return string(so), string(se)
}

// fakeDaemon points dial at an in-process handler. It records requests.
type fakeDaemon struct {
	mu   sync.Mutex
	reqs []common.Request
}

func (f *fakeDaemon) methods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var m []string
	for _, r := range f.reqs {
		m = append(m, r.Method)
	}
	return m
}

func (f *fakeDaemon) last() common.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reqs[len(f.reqs)-1]
}

func useFake(t *testing.T, handler func(common.Request) common.Response) *fakeDaemon {
	t.Helper()
	f := &fakeDaemon{}
	oldDial, oldSpawn := dial, spawn
	dial = func() (net.Conn, error) {
		c, s := net.Pipe()
		go func() {
			defer s.Close()
			line, err := bufio.NewReader(s).ReadBytes('\n')
			if err != nil {
				return
			}
			var req common.Request
			if json.Unmarshal(line, &req) != nil {
				return
			}
			f.mu.Lock()
			f.reqs = append(f.reqs, req)
			f.mu.Unlock()
			b, _ := json.Marshal(handler(req))
			s.Write(append(b, '\n'))
		}()
		return c, nil
	}
	spawn = func() error { t.Error("unexpected daemon spawn"); return nil }
	t.Cleanup(func() { dial, spawn = oldDial, oldSpawn })
	return f
}

func okData(v any) common.Response {
	b, _ := json.Marshal(v)
	return common.Response{OK: true, Data: b}
}

func sampleInfos() []common.ProcessInfo {
	return []common.ProcessInfo{{ID: 0, Name: "web", Status: "online", PID: 42, UptimeMs: 90_000, MemoryB: 3 << 20, CPU: 1.5}}
}

func contains(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("output missing %q:\n%s", w, got)
		}
	}
}

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
