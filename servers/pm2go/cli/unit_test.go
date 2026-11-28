package cli

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

func TestParseInterspersed(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		pos      []string
		rest     []string
		name_    string
		envs     []string
		wantFail bool
	}{
		{name: "flags before", args: []string{"--name", "a", "app.js"}, pos: []string{"app.js"}, name_: "a"},
		{name: "flags after", args: []string{"app.js", "--name", "a"}, pos: []string{"app.js"}, name_: "a"},
		{name: "flags between", args: []string{"x", "--name", "a", "y"}, pos: []string{"x", "y"}, name_: "a"},
		{name: "double dash", args: []string{"app.js", "--name", "a", "--", "--port", "80", "-v"}, pos: []string{"app.js"}, rest: []string{"--port", "80", "-v"}, name_: "a"},
		{name: "double dash first", args: []string{"--", "a", "b"}, rest: []string{"a", "b"}},
		{name: "repeated env", args: []string{"--env", "A=1", "app", "--env", "B=2"}, pos: []string{"app"}, envs: []string{"A=1", "B=2"}},
		{name: "empty", args: nil},
		{name: "unknown flag", args: []string{"--nope"}, wantFail: true},
		{name: "unknown flag after dash", args: []string{"a", "--nope", "--", "b"}, wantFail: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs, o := newStartFlagSet()
			fs.SetOutput(os.NewFile(0, os.DevNull))
			pos, rest, err := parseInterspersed(fs, c.args)
			if c.wantFail {
				if err == nil {
					t.Fatal("want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(pos, c.pos) {
				t.Errorf("pos = %v, want %v", pos, c.pos)
			}
			if !reflect.DeepEqual(rest, c.rest) {
				t.Errorf("rest = %v, want %v", rest, c.rest)
			}
			if o.name != c.name_ {
				t.Errorf("name = %q, want %q", o.name, c.name_)
			}
			if !reflect.DeepEqual([]string(o.envs), c.envs) {
				t.Errorf("envs = %v, want %v", o.envs, c.envs)
			}
		})
	}
}

func TestMultiString(t *testing.T) {
	m := multi{"a", "b"}
	if m.String() != "a,b" {
		t.Fatal(m.String())
	}
}

func parseStart(t *testing.T, args ...string) (*startOpts, string, []string) {
	t.Helper()
	fs, o := newStartFlagSet()
	pos, rest, err := parseInterspersed(fs, args)
	if err != nil || len(pos) != 1 {
		t.Fatalf("parse: pos=%v err=%v", pos, err)
	}
	return o, pos[0], rest
}

func TestBuildStartConfigFlags(t *testing.T) {
	dir := t.TempDir()
	o, target, rest := parseStart(t, "node server.js", "--cwd", dir, "--interpreter", "bash", "--namespace", "ns",
		"--max-restarts", "4", "--min-uptime", "100", "--restart-delay", "200", "--exp-backoff-restart-delay", "300",
		"--kill-timeout", "400", "--instances", "3", "--watch", "--ignore-watch", "a, b,,c", "--cron-restart", "0 4 * * *",
		"--max-memory-restart", "512", "--log-max-size", "-1", "--log-retain", "7", "--stop-exit-codes", "0, 2",
		"--no-autorestart", "--env", "A=1", "--env", "B=x=y", "--", "--port", "80")
	c, err := buildStartConfig(o, target, rest)
	if err != nil {
		t.Fatal(err)
	}
	f := false
	want := common.AppConfig{
		Name: "node", Script: "node server.js", Args: []string{"--port", "80"}, Cwd: dir, Interpreter: "bash",
		Namespace: "ns", MaxRestarts: 4, MinUptimeMs: 100, RestartDelayMs: 200, ExpBackoffRestartDelayMs: 300,
		KillTimeoutMs: 400, Instances: 3, Watch: true, IgnoreWatch: []string{"a", "b", "c"}, CronRestart: "0 4 * * *",
		MaxMemoryRestartMB: 512, LogMaxSizeMB: -1, LogRetain: 7, StopExitCodes: []int{0, 2}, Autorestart: &f,
		Env: map[string]string{"A": "1", "B": "x=y"},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("got  %+v\nwant %+v", c, want)
	}
}

func TestBuildStartConfigDefaults(t *testing.T) {
	o, target, rest := parseStart(t, "/opt/bin/worker.py")
	c, err := buildStartConfig(o, target, rest)
	if err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	if c.Name != "worker" || c.Cwd != wd || c.Autorestart != nil || c.Env != nil || c.StopExitCodes != nil || c.IgnoreWatch != nil {
		t.Errorf("unexpected defaults: %+v", c)
	}
}

func TestBuildStartConfigRelativeCwd(t *testing.T) {
	o, target, rest := parseStart(t, "x", "--cwd", "sub/dir")
	c, err := buildStartConfig(o, target, rest)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(c.Cwd) || !strings.HasSuffix(c.Cwd, "sub/dir") {
		t.Errorf("cwd = %q", c.Cwd)
	}
}

func TestBuildStartConfigErrors(t *testing.T) {
	for _, tc := range []struct{ name, flag, val, msg string }{
		{"env no equals", "--env", "NOVALUE", `bad --env "NOVALUE"`},
		{"env empty key", "--env", "=v", `bad --env "=v"`},
		{"stop codes", "--stop-exit-codes", "0,x", `bad --stop-exit-codes value "x"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, target, rest := parseStart(t, "app", tc.flag, tc.val)
			_, err := buildStartConfig(o, target, rest)
			if err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("err = %v, want %q", err, tc.msg)
			}
		})
	}
}

func writeEco(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ecosystem.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveStartTargets(t *testing.T) {
	eco := writeEco(t, `{"apps":[{"name":"a","script":"sleep 1","env":{"K":"base"}},{"name":"b","script":"sleep 2"},{"script":"sleep 3"}]}`)

	o, _, _ := parseStart(t, "x")
	cfgs, err := resolveStartTargets(o, eco, nil)
	if err != nil || len(cfgs) != 3 {
		t.Fatalf("all: %v %v", cfgs, err)
	}
	if cfgs[2].Name != "sleep" {
		t.Errorf("default name = %q", cfgs[2].Name)
	}

	o, _, _ = parseStart(t, "x", "--only", "b, sleep")
	cfgs, err = resolveStartTargets(o, eco, nil)
	if err != nil || len(cfgs) != 2 || cfgs[0].Name != "b" || cfgs[1].Name != "sleep" {
		t.Fatalf("only: %v %v", cfgs, err)
	}

	o, _, _ = parseStart(t, "x", "--only", "zzz")
	if _, err = resolveStartTargets(o, eco, nil); err == nil || !strings.Contains(err.Error(), "no app matches --only") {
		t.Fatalf("no match: %v", err)
	}

	o, _, _ = parseStart(t, "x", "--env-profile", "prod")
	if _, err = resolveStartTargets(o, eco, nil); err == nil {
		t.Fatal("missing profile must fail")
	}

	o, _, _ = parseStart(t, "x")
	if _, err = resolveStartTargets(o, filepath.Join(t.TempDir(), "none.json"), nil); err == nil {
		t.Fatal("missing file must fail")
	}

	// upper-case extension still counts as an ecosystem file
	up := filepath.Join(t.TempDir(), "E.JSON")
	os.WriteFile(up, []byte(`{"apps":[{"name":"u","script":"sleep 1"}]}`), 0o644)
	if cfgs, err = resolveStartTargets(o, up, nil); err != nil || cfgs[0].Name != "u" {
		t.Fatalf("upper: %v %v", cfgs, err)
	}
}

func TestResolveStartTargetsCommand(t *testing.T) {
	o, target, rest := parseStart(t, "sleep", "--name", "z", "--", "5")
	cfgs, err := resolveStartTargets(o, target, rest)
	if err != nil || len(cfgs) != 1 || cfgs[0].Name != "z" || !reflect.DeepEqual(cfgs[0].Args, []string{"5"}) {
		t.Fatalf("%+v %v", cfgs, err)
	}
	o, target, rest = parseStart(t, "sleep", "--env", "bad")
	if _, err = resolveStartTargets(o, target, rest); err == nil {
		t.Fatal("want error")
	}
}

func TestRenderUnit(t *testing.T) {
	d := renderUnit("daemon", "/usr/bin/pm2go", "alice", "/home/alice/.pm2go", nil)
	contains(t, d, "Description=pm2go daemon", "User=alice", "Environment=PMGO_HOME=/home/alice/.pm2go",
		"ExecStart=/usr/bin/pm2go daemon --resurrect\n", "Type=simple", "KillMode=process", "WantedBy=multi-user.target")
	if strings.Contains(d, "ExecStop") {
		t.Error("ExecStop would also run on restart and stop the applications")
	}
	h := renderUnit("hub", "/usr/bin/pm2go", "bob", "/var/hub", []string{"--listen", ":9000"})
	contains(t, h, "KillMode=control-group")
	contains(t, h, "Description=pm2go hub", "User=bob", "ExecStart=/usr/bin/pm2go hub --listen :9000\n")
	if strings.Contains(h, "--resurrect") {
		t.Error("hub unit must not resurrect")
	}
	h2 := renderUnit("hub", "/x", "u", "/h", nil)
	contains(t, h2, "ExecStart=/x hub\n")
	if unitName("hub") != "pm2go-hub.service" {
		t.Error(unitName("hub"))
	}
}

func TestCmdStartup(t *testing.T) {
	if err := cmdStartup([]string{"bogus"}, true); err == nil || !strings.Contains(err.Error(), "unknown target") {
		t.Fatalf("err = %v", err)
	}
	if os.Geteuid() == 0 {
		t.Skip("would write a real unit as root")
	}
	var err error
	out, _ := capture(t, func() { err = cmdStartup([]string{"hub", "--listen", ":1"}, true) })
	if err == nil || !strings.Contains(err.Error(), "sudo") {
		t.Fatalf("err = %v", err)
	}
	contains(t, out, "Not root", "pm2go-hub.service", "hub --listen :1")
}

func TestHumanDuration(t *testing.T) {
	for in, want := range map[int64]string{
		-5: "0s", 0: "0s", 999: "0s", 1000: "1s", 59_000: "59s", 60_000: "1m", 3_599_000: "59m",
		3_600_000: "1h", 47 * 3_600_000: "47h", 48 * 3_600_000: "2d", 100 * 24 * 3_600_000: "100d",
	} {
		if got := humanDuration(in); got != want {
			t.Errorf("humanDuration(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	for in, want := range map[int64]string{
		-1: "0b", 0: "0b", 1: "1b", 1023: "1023b", 1024: "1.0kb", 1536: "1.5kb", 1 << 20: "1.0mb",
		5 << 20: "5.0mb", 1 << 30: "1.0gb", 3 << 30: "3.0gb",
	} {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestPrintTable(t *testing.T) {
	var sb strings.Builder
	printTable(&sb, nil)
	if sb.String() != "no processes\n" {
		t.Errorf("empty: %q", sb.String())
	}
	sb.Reset()
	infos := sampleInfos()
	infos = append(infos, common.ProcessInfo{ID: 1, Name: "api", Namespace: "prod", Status: "errored"})
	printTable(&sb, infos)
	contains(t, sb.String(), "ID", "NAMESPACE", "web", "default", "1m", "online", "1.5%", "3.0mb", "api", "prod", "errored")
}

func TestPrintDescribe(t *testing.T) {
	code := 3
	af := false
	i := common.ProcessInfo{Name: "web", ID: 2, Status: "stopped", PID: 9, ExitCode: &code, OutLog: "/o.log", ErrLog: "/e.log",
		CreatedAt: 1_700_000_000_000, Config: common.AppConfig{Script: "sleep", Args: []string{"1", "2"}, Cwd: "/w", Autorestart: &af}}
	var sb strings.Builder
	printDescribe(&sb, i)
	contains(t, sb.String(), "name", "web", "last exit code", "3", "sleep", "1 2", "/o.log", "/e.log", "autorestart", "false", "kill timeout")
	i.ExitCode = nil
	sb.Reset()
	printDescribe(&sb, i)
	if strings.Contains(sb.String(), "last exit code") {
		t.Error("exit code row must be hidden")
	}
}

func TestLastLines(t *testing.T) {
	dir := t.TempDir()
	w := func(name, body string) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte(body), 0o644)
		return p
	}
	if ls, off := lastLines(filepath.Join(dir, "missing"), 5); ls != nil || off != 0 {
		t.Errorf("missing: %v %d", ls, off)
	}
	if ls, off := lastLines(w("empty", ""), 5); ls != nil || off != 0 {
		t.Errorf("empty: %v %d", ls, off)
	}
	if ls, off := lastLines(w("nl", "\n\n"), 5); ls != nil || off != 2 {
		t.Errorf("only newlines: %v %d", ls, off)
	}
	if ls, off := lastLines(w("small", "a\nb\nc\n"), 2); !reflect.DeepEqual(ls, []string{"b", "c"}) || off != 6 {
		t.Errorf("small: %v %d", ls, off)
	}
	if ls, _ := lastLines(w("fewer", "a\nb"), 10); !reflect.DeepEqual(ls, []string{"a", "b"}) {
		t.Errorf("fewer: %v", ls)
	}
	// larger than the 1 MiB read window: only the tail is read
	var sb strings.Builder
	for i := 0; sb.Len() < 3<<20; i++ {
		sb.WriteString("line-0123456789\n")
	}
	sb.WriteString("last-one\n")
	big := w("big", sb.String())
	ls, off := lastLines(big, 3)
	if len(ls) != 3 || ls[2] != "last-one" || off != int64(sb.Len()) {
		t.Errorf("big: %v %d (want off %d)", ls, off, sb.Len())
	}
}

func TestShowLogsNoFollow(t *testing.T) {
	dir := t.TempDir()
	o, e := filepath.Join(dir, "o.log"), filepath.Join(dir, "e.log")
	os.WriteFile(o, []byte("o1\no2\n"), 0o644)
	os.WriteFile(e, []byte("e1\n"), 0o644)
	infos := []common.ProcessInfo{{Name: "web", OutLog: o, ErrLog: e}}
	out, _ := capture(t, func() { showLogs(infos, 1, false, false, false) })
	contains(t, out, "web | out: o2", "web | err: e1")
	if strings.Contains(out, "o1") {
		t.Error("line limit ignored")
	}
	out, _ = capture(t, func() { showLogs(infos, 5, false, true, false) })
	if strings.Contains(out, "err") {
		t.Errorf("--out shows stderr: %s", out)
	}
	out, _ = capture(t, func() { showLogs(infos, 5, false, false, true) })
	if strings.Contains(out, "| out") {
		t.Errorf("--err shows stdout: %s", out)
	}
}

func TestFollowFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "o.log")
	os.WriteFile(p, []byte("first\n"), 0o644)
	src := &logSource{label: "w | out", path: p}
	_, src.off = lastLines(p, 5)
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	done := make(chan struct{})
	out, _ := capture(t, func() {
		go func() { followFile(ctx, src, &mu); close(done) }()
		time.Sleep(50 * time.Millisecond)
		f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
		f.WriteString("second\npart")
		time.Sleep(500 * time.Millisecond)
		f.WriteString("ial\n")
		f.Close()
		time.Sleep(500 * time.Millisecond)
		os.WriteFile(p, []byte("after-truncate\n"), 0o644) // shorter than offset
		time.Sleep(500 * time.Millisecond)
		cancel()
		<-done
	})
	contains(t, out, "w | out: second", "w | out: partial", "w | out: after-truncate")
	if strings.Contains(out, "first") {
		t.Errorf("replayed old line: %s", out)
	}
}

func TestCmdKillAbsent(t *testing.T) {
	old := dial
	dial = func() (net.Conn, error) { return nil, errors.New("refused") }
	defer func() { dial = old }()
	var err error
	out, _ := capture(t, func() { err = cmdKill() })
	if err != nil || !strings.Contains(out, "daemon is not running") {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if code := mainCode(t, "kill"); code != 0 {
		t.Errorf("exit %d", code)
	}
}

func mainCode(t *testing.T, args ...string) (code int) {
	t.Helper()
	capture(t, func() { code = Main(args) })
	return
}

func TestMainExitCodes(t *testing.T) {
	_, errOut := capture(t, func() {
		if c := Main([]string{"frobnicate"}); c != 2 {
			t.Errorf("unknown exit = %d", c)
		}
	})
	contains(t, errOut, `unknown command "frobnicate"`, "Usage: pm2go")
	for _, a := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
		o, _ := capture(t, func() {
			if c := Main(a); c != 0 {
				t.Errorf("%v exit = %d", a, c)
			}
		})
		contains(t, o, "Usage: pm2go", "start flags:")
	}
	// argument errors exit 1 with an error line, no daemon needed
	old, oldSpawn := dial, spawn
	dial = func() (net.Conn, error) { t.Error("dial reached"); return nil, errors.New("x") }
	spawn = func() error { t.Error("spawn reached"); return nil }
	defer func() { dial, spawn = old, oldSpawn }()
	for _, a := range [][]string{
		{"start"}, {"start", "a", "b"}, {"start", "app", "--env", "bad"}, {"start", "--nope"},
		{"stop"}, {"restart", "a", "b"}, {"delete"}, {"reload"}, {"scale", "a"}, {"scale", "a", "x"},
		{"describe"}, {"link"}, {"link", "--hub", "http://h"}, {"startup", "bogus"}, {"unstartup", "bogus"},
	} {
		_, e := capture(t, func() {
			if c := Main(a); c != 1 {
				t.Errorf("%v exit = %d, want 1", a, c)
			}
		})
		if !strings.Contains(e, "error:") {
			t.Errorf("%v stderr = %q", a, e)
		}
	}
}

func TestCommandsAgainstFake(t *testing.T) {
	shortHome(t)
	f := useFake(t, func(r common.Request) common.Response {
		switch r.Method {
		case common.MethodSave:
			return okData(map[string]int{"saved": 2})
		case common.MethodPing:
			return okData(map[string]int{"pid": 77})
		case common.MethodDescribe, common.MethodList, common.MethodResurrect, common.MethodStart,
			"stop", "restart", "delete", "reload":
			return okData(sampleInfos())
		case common.MethodFlush, common.MethodScale, common.MethodLink:
			return common.Response{OK: true}
		}
		return common.Response{Error: "boom " + r.Method}
	})
	run := func(args ...string) (int, string, string) {
		var code int
		o, e := capture(t, func() { code = Main(args) })
		return code, o, e
	}
	if c, o, _ := run("ls"); c != 0 {
		t.Fatal(c)
	} else {
		contains(t, o, "web", "online")
	}
	run("list")
	run("status")
	if c, o, _ := run("jlist"); c != 0 {
		t.Fatal(c)
	} else {
		contains(t, o, `"name": "web"`)
	}
	if _, o, _ := run("save"); !strings.Contains(o, "saved 2 process(es)") {
		t.Errorf("save: %q", o)
	}
	if _, o, _ := run("ping"); !strings.Contains(o, "pid 77") {
		t.Errorf("ping: %q", o)
	}
	if _, o, _ := run("resurrect"); !strings.Contains(o, "web") {
		t.Errorf("resurrect: %q", o)
	}
	for _, m := range []string{"stop", "restart", "delete", "reload"} {
		if c, _, _ := run(m, "web"); c != 0 {
			t.Errorf("%s exit %d", m, c)
		}
	}
	if c, o, _ := run("describe", "web"); c != 0 || !strings.Contains(o, "script") {
		t.Errorf("describe: %d %q", c, o)
	}
	if c, _, _ := run("show", "web"); c != 0 {
		t.Errorf("show: %d", c)
	}
	if c, _, _ := run("flush"); c != 0 {
		t.Errorf("flush: %d", c)
	}
	if c, _, _ := run("flush", "web"); c != 0 {
		t.Errorf("flush web: %d", c)
	}
	if c, _, _ := run("scale", "web", "3"); c != 0 {
		t.Errorf("scale: %d", c)
	}
	if c, o, _ := run("logs", "web", "--nostream", "--lines", "3"); c != 0 {
		t.Errorf("logs: %d %q", c, o)
	}
	if c, _, _ := run("logs", "--nostream"); c != 0 {
		t.Errorf("logs all: %d", c)
	}
	if c, _, _ := run("start", "sleep", "--name", "s", "--", "9"); c != 0 {
		t.Errorf("start: %d", c)
	}
	// the start request carries the built config and the caller environment
	var sp common.StartParams
	for _, r := range f.reqs {
		if r.Method == common.MethodStart {
			if err := jsonUnmarshal(r.Params, &sp); err != nil {
				t.Fatal(err)
			}
		}
	}
	if sp.Config.Name != "s" || !reflect.DeepEqual(sp.Config.Args, []string{"9"}) || len(sp.CLIEnv) == 0 {
		t.Errorf("start params: %+v", sp.Config)
	}
	// link saves the config and tells the daemon
	if c, o, _ := run("link", "--hub", "http://h:1", "--key", "k", "--name", "m"); c != 0 || !strings.Contains(o, "linked to http://h:1") {
		t.Errorf("link: %d %q", c, o)
	}
	l, err := common.LoadLink()
	if err != nil || l.HubURL != "http://h:1" || l.Key != "k" || l.Machine != "m" {
		t.Errorf("saved link: %+v %v", l, err)
	}
	if c, o, _ := run("unlink"); c != 0 || !strings.Contains(o, "unlinked") {
		t.Errorf("unlink: %d %q", c, o)
	}
	if _, err := os.Stat(common.LinkFile()); !os.IsNotExist(err) {
		t.Errorf("link file remains: %v", err)
	}
	if c, _, _ := run("unlink"); c != 0 { // already unlinked is fine
		t.Errorf("second unlink: %d", c)
	}
	// daemon error surfaces as exit 1
	useFake(t, func(r common.Request) common.Response { return common.Response{Error: "nope"} })
	if c, _, e := run("stop", "web"); c != 1 || !strings.Contains(e, "error: nope") {
		t.Errorf("stop err: %d %q", c, e)
	}
	if c, _, e := run("start", "sleep"); c != 1 || !strings.Contains(e, "sleep: nope") {
		t.Errorf("start err: %d %q", c, e)
	}
	for _, a := range []string{"ls", "jlist", "save", "ping", "resurrect", "flush", "kill"} {
		if c, _, _ := run(a); c != 1 {
			t.Errorf("%s with daemon error: exit %d", a, c)
		}
	}
	if c, _, _ := run("logs", "--nostream"); c != 1 {
		t.Errorf("logs err: %d", c)
	}
	if c, _, _ := run("describe", "x"); c != 1 {
		t.Errorf("describe err: %d", c)
	}
	if c, _, _ := run("scale", "x", "1"); c != 1 {
		t.Errorf("scale err: %d", c)
	}
}

func TestStartEcosystemSendsEachApp(t *testing.T) {
	f := useFake(t, func(r common.Request) common.Response { return okData(sampleInfos()) })
	eco := writeEco(t, `{"apps":[{"name":"a","script":"sleep 1"},{"name":"b","script":"sleep 2"}]}`)
	if code := mainCode(t, "start", eco, "--only", "b"); code != 0 {
		t.Fatal(code)
	}
	n := 0
	for _, m := range f.methods() {
		if m == common.MethodStart {
			n++
		}
	}
	if n != 1 {
		t.Errorf("start requests = %d, want 1", n)
	}
	if code := mainCode(t, "start", eco, "--only", "none"); code != 1 {
		t.Errorf("no match exit = %d", code)
	}
}

func TestRoundTripErrors(t *testing.T) {
	// unmarshalable params
	c, s := net.Pipe()
	defer c.Close()
	defer s.Close()
	if err := roundTrip(c, "x", make(chan int), nil); err == nil {
		t.Error("want marshal error")
	}
	// closed connection
	c2, s2 := net.Pipe()
	s2.Close()
	if err := roundTrip(c2, "x", nil, nil); err == nil {
		t.Error("want write error")
	}
	// garbage reply
	c3, s3 := net.Pipe()
	go func() { buf := make([]byte, 256); s3.Read(buf); s3.Write([]byte("not json\n")); s3.Close() }()
	if err := roundTrip(c3, "x", map[string]int{"a": 1}, nil); err == nil {
		t.Error("want decode error")
	}
	// EOF before reply
	c4, s4 := net.Pipe()
	go func() { buf := make([]byte, 256); s4.Read(buf); s4.Close() }()
	if err := roundTrip(c4, "x", nil, nil); err == nil {
		t.Error("want read error")
	}
	// data of the wrong shape
	c5, s5 := net.Pipe()
	go func() {
		buf := make([]byte, 256)
		s5.Read(buf)
		s5.Write([]byte(`{"ok":true,"data":"str"}` + "\n"))
		s5.Close()
	}()
	var m map[string]int
	if err := roundTrip(c5, "x", nil, &m); err == nil {
		t.Error("want unmarshal error")
	}
}

func TestConnectSpawn(t *testing.T) {
	oldDial, oldSpawn, oldWait := dial, spawn, spawnWait
	defer func() { dial, spawn, spawnWait = oldDial, oldSpawn, oldWait }()

	// no spawn requested
	dial = func() (net.Conn, error) { return nil, errors.New("refused") }
	spawn = func() error { t.Error("spawned"); return nil }
	if _, err := connect(false); err == nil || err.Error() != "daemon is not running" {
		t.Errorf("err = %v", err)
	}

	// spawn succeeds, daemon appears on the third dial
	shortHome(t)
	spawned, dials := 0, 0
	dial = func() (net.Conn, error) {
		dials++
		if dials < 3 {
			return nil, errors.New("refused")
		}
		c, s := net.Pipe()
		s.Close()
		return c, nil
	}
	spawn = func() error { spawned++; return nil }
	conn, err := connect(true)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if spawned != 1 {
		t.Errorf("spawned %d times", spawned)
	}

	// spawn fails
	dial = func() (net.Conn, error) { return nil, errors.New("refused") }
	spawn = func() error { return errors.New("no exe") }
	if _, err := connect(true); err == nil || err.Error() != "no exe" {
		t.Errorf("err = %v", err)
	}

	// daemon never comes up
	spawn = func() error { return nil }
	spawnWait = 120 * time.Millisecond
	if _, err := connect(true); err == nil || !strings.Contains(err.Error(), "daemon did not start, see") {
		t.Errorf("err = %v", err)
	}
}

func TestDefaultDial(t *testing.T) {
	home := shortHome(t)
	if _, err := defaultDial(); err == nil {
		t.Fatal("want error without a socket")
	}
	ln, err := net.Listen("unix", filepath.Join(home, "rpc.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		if c, err := ln.Accept(); err == nil {
			c.Close()
		}
	}()
	c, err := defaultDial()
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
}

func TestSpawnDaemonFailsWithoutLogDir(t *testing.T) {
	// PMGO_HOME below a regular file: EnsureDirs cannot create it.
	f := filepath.Join(t.TempDir(), "file")
	os.WriteFile(f, nil, 0o644)
	t.Setenv("PMGO_HOME", filepath.Join(f, "home"))
	if err := spawnDaemon(); err == nil {
		t.Fatal("want error")
	}
}
