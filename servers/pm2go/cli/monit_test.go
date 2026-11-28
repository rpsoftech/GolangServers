package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

func monitInfos(n int) []common.ProcessInfo {
	var out []common.ProcessInfo
	for i := 0; i < n; i++ {
		st := common.StatusOnline
		if i%4 == 3 {
			st = common.StatusErrored
		}
		out = append(out, common.ProcessInfo{ID: i, Name: fmt.Sprintf("app-%d", i), Status: st,
			CPU: float64(i * 13 % 100), MemoryB: int64(i+1) << 24})
	}
	return out
}

func monitSized(n int) *monitState {
	s := &monitState{selID: -1, haveSys: true, noColor: true}
	s.sys = common.SystemSample{CPU: 12, MemUsed: 2 << 30, MemTotal: 8 << 30, MemPercent: 25, Load1: 0.5, UptimeS: 90000}
	s.update(monitInfos(n))
	s.logs = []string{"hello world", "\x1b[31mred\x1b[0m\tline", strings.Repeat("x", 500)}
	return s
}

func checkGeometry(t *testing.T, out string, w, h int) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != h {
		t.Fatalf("%dx%d: got %d lines", w, h, len(lines))
	}
	for n, l := range lines {
		if got := utf8.RuneCountInString(stripANSI(l)); got != w {
			t.Fatalf("%dx%d line %d is %d cells: %q", w, h, n, got, l)
		}
	}
}

func TestRenderSizes(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {200, 60}, {100, 10}, {40, 8}, {30, 6}} {
		for _, n := range []int{0, 3, 40} {
			s := monitSized(n)
			checkGeometry(t, render(s, sz[0], sz[1]), sz[0], sz[1])
			s.noColor = false
			checkGeometry(t, render(s, sz[0], sz[1]), sz[0], sz[1])
		}
	}
}

func TestRenderContent(t *testing.T) {
	s := monitSized(3)
	out := render(s, 80, 24)
	contains(t, out, "pm2go monit", "3/3 online", "cpu 12%", "mem 25% (2.0G/8.0G)", "PROCESSES 3", "app-0", "app-2",
		"LOGS app-0 [out]", "hello world", "red    line", "q quit")
	if strings.Contains(out, "\x1b") || strings.Contains(out, "\t") {
		t.Error("noColor render must not contain escapes or tabs")
	}
	wide := render(s, 200, 60)
	contains(t, wide, "│", "██")
	s.errLog = true
	contains(t, render(s, 80, 24), "[err]")
}

func TestRenderColorAndTooSmall(t *testing.T) {
	s := monitSized(2)
	s.noColor = false
	contains(t, render(s, 80, 24), "\x1b[32m", "\x1b[7m")
	out := render(s, 20, 5)
	contains(t, out, "too small")
	if strings.Count(out, "\n") != 0 {
		t.Error("too-small message is one line")
	}
}

func TestRenderSelectionScrolls(t *testing.T) {
	s := monitSized(40)
	for i := 0; i < 39; i++ {
		s.key("down")
	}
	contains(t, render(s, 80, 24), "app-39")
}

func TestRenderPaneStates(t *testing.T) {
	s := &monitState{selID: -1, noColor: true}
	contains(t, render(s, 80, 24), "no processes", "LOGS")
	s.update(monitInfos(2))
	s.filter = "zzz"
	contains(t, render(s, 80, 24), "no match")
	s.setStatus("boom", true)
	contains(t, render(s, 80, 24), "error: boom")
}

func TestKeysNavigationAndActions(t *testing.T) {
	s := monitSized(3)
	s.key("down")
	s.key("j")
	s.key("down") // clamped
	if s.selID != 2 {
		t.Fatalf("selID %d", s.selID)
	}
	s.key("up")
	s.key("k")
	s.key("k")
	if s.selID != 0 {
		t.Fatalf("selID %d", s.selID)
	}
	s.key("down")
	for k, want := range map[string]string{"r": common.MethodRestart, "s": common.MethodStop, "f": common.MethodFlush} {
		a := s.key(k)
		if a.method != want || a.target != "1" {
			t.Errorf("%s -> %+v", k, a)
		}
	}
	if a := s.key("e"); !s.errLog || !a.redraw {
		t.Error("e toggles err log")
	}
	if !s.key("q").quit || !s.key("ctrl-c").quit {
		t.Error("q and ctrl-c quit")
	}
	if a := s.key("x"); a != (monitAction{}) {
		t.Errorf("unknown key: %+v", a)
	}
}

func TestKeysDeleteConfirm(t *testing.T) {
	s := monitSized(2)
	if a := s.key("d"); a.method != "" || s.confirm == "" {
		t.Fatal("d must only ask")
	}
	contains(t, render(s, 80, 24), "delete app-0? y/n")
	if a := s.key("n"); a.method != "" || s.confirm != "" {
		t.Fatal("n cancels")
	}
	s.key("d")
	if a := s.key("y"); a.method != common.MethodDelete || a.target != "0" {
		t.Fatalf("y deletes: %+v", a)
	}
	s.key("d")
	if !s.key("ctrl-c").quit {
		t.Error("ctrl-c quits while confirming")
	}
}

func TestKeysFilter(t *testing.T) {
	s := monitSized(12)
	s.key("/")
	for _, k := range []string{"a", "p", "p", "-", "1"} {
		s.key(k)
	}
	if got := len(s.visible()); got != 3 {
		t.Fatalf("visible %d", got)
	}
	if s.selID != 1 {
		t.Fatalf("selection follows filter: %d", s.selID)
	}
	s.key("backspace")
	s.key("enter")
	if s.filtering || s.filter != "app-" {
		t.Fatalf("filter %q filtering %v", s.filter, s.filtering)
	}
	if a := s.key("q"); !a.quit {
		t.Error("q quits after filter applied")
	}
	s.key("esc")
	if s.filter != "" {
		t.Error("esc clears filter")
	}
	s.key("/")
	if a := s.key("q"); a.quit || s.filter != "q" {
		t.Error("q is text while filtering")
	}
	s.key("esc")
	if s.filtering || s.filter != "" {
		t.Error("esc cancels filtering")
	}
}

func TestKeysNoSelection(t *testing.T) {
	s := &monitState{selID: -1, noColor: true}
	if a := s.key("r"); a.method != "" || !s.statusErr {
		t.Errorf("restart without selection: %+v", a)
	}
}

func TestParseKeys(t *testing.T) {
	cases := map[string][]string{
		"\x1b[A":     {"up"},
		"\x1bOB":     {"down"},
		"\x1b":       {"esc"},
		"q":          {"q"},
		"jk":         {"j", "k"},
		"\x03":       {"ctrl-c"},
		"\r":         {"enter"},
		"\x7f":       {"backspace"},
		"\x1b[1;5C/": {"/"},
		"é":          {"é"},
	}
	for in, want := range cases {
		got := parseKeys([]byte(in))
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%q -> %v want %v", in, got, want)
		}
	}
}

func TestFrameAndFit(t *testing.T) {
	f := frame("ab\ncd")
	if !strings.HasPrefix(f, "\x1b[H") || strings.Count(f, "\r\n") != 1 || !strings.HasSuffix(f, "\x1b[J") {
		t.Errorf("frame %q", f)
	}
	if fit("abcdef", 4) != "abc…" || fit("ab", 4) != "ab  " || fit("x", 0) != "" || fit("ab", 1) != "a" {
		t.Error("fit")
	}
	if clean("\x1b[1mA\x1b[0m\tB\x00") != "A    B" {
		t.Errorf("clean %q", clean("\x1b[1mA\x1b[0m\tB\x00"))
	}
}

func TestMonitNoTTY(t *testing.T) {
	// go test stdin/stdout are not terminals
	if err := cmdMonit(nil); err != errNoTTY {
		t.Fatalf("got %v", err)
	}
	if err := cmdMonit([]string{"x"}); err == nil {
		t.Error("args rejected")
	}
	_, errOut := capture(t, func() {
		if code := Main([]string{"monit"}); code != 1 {
			t.Errorf("exit %d", code)
		}
	})
	contains(t, errOut, "needs an interactive terminal")
}

func TestMonitRPCDaemonDown(t *testing.T) {
	shortHome(t)
	s := &monitState{selID: -1, noColor: true}
	s.refresh()
	if !s.statusErr || !strings.Contains(s.status, "daemon is not running") {
		t.Errorf("status %q", s.status)
	}
}

// syncBuf is a goroutine-safe buffer for the loop's output.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestRunMonitLoop(t *testing.T) {
	dir := t.TempDir()
	outLog := filepath.Join(dir, "out.log")
	os.WriteFile(outLog, []byte("line one\nline two\n"), 0o644)
	infos := []common.ProcessInfo{
		{ID: 0, Name: "web", Status: "online", OutLog: outLog},
		{ID: 1, Name: "db", Status: "online"},
	}
	fd := useFake(t, func(r common.Request) common.Response {
		switch r.Method {
		case common.MethodList:
			return okData(infos)
		case common.MethodSystem:
			return okData(common.SystemSample{CPU: 7, CPUCount: 2})
		case common.MethodRestart:
			return common.Response{Error: "cannot restart"}
		}
		return okData(infos)
	})
	keys := make(chan []byte)
	tick := make(chan time.Time)
	winch := make(chan os.Signal, 1)
	quit := make(chan os.Signal, 1)
	out := &syncBuf{}
	finished := make(chan struct{})
	st := &monitState{selID: -1, noColor: true}
	go func() {
		runMonit(st, monitIO{keys: keys, out: out, size: func() (int, int) { return 80, 24 }, winch: winch, quit: quit, tick: tick})
		close(finished)
	}()
	waitFor := func(sub string) {
		t.Helper()
		for i := 0; i < 200; i++ {
			if strings.Contains(out.String(), sub) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("never saw %q in:\n%s", sub, out.String())
	}
	waitFor("line two")
	waitFor("cpu 7%")
	keys <- []byte("r")
	waitFor("error: cannot restart")
	if fd.last().Method != common.MethodList && !strings.Contains(strings.Join(fd.methods(), ","), "restart") {
		t.Error("restart RPC not sent")
	}
	winch <- nil
	tick <- time.Now()
	keys <- []byte("\x1b[B")
	waitFor("LOGS db")
	keys <- []byte("q")
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not stop on q")
	}
	// quit signal path
	finished = make(chan struct{})
	go func() {
		runMonit(&monitState{selID: -1, noColor: true}, monitIO{keys: keys, out: &syncBuf{}, size: func() (int, int) { return 80, 24 }, winch: winch, quit: quit, tick: tick})
		close(finished)
	}()
	quit <- os.Interrupt
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not stop on signal")
	}
}
