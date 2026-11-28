package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// Proc is one supervised process. A single goroutine (loop) owns the child
// lifecycle; Stop signals it and waits.
type Proc struct {
	mu       sync.Mutex
	m        *Manager
	id       int
	instance int
	cfg      common.AppConfig
	env      []string // base environment for the child

	status    string
	pid       int
	restarts  int
	unstable  int
	startedAt time.Time
	createdAt time.Time
	exitCode  *int
	use       usage

	stopping   bool
	stopCh     chan struct{}
	done       chan struct{} // non-nil while loop runs
	restartReq chan string
	reloadReq  chan chan error // overlapping reload requests, served by supervise
	cancel     context.CancelFunc

	outF, errF *os.File // log files the child writes to; nil for an adopted child until it respawns
	outT, errT *logTail // follow the files and feed the log hook

	pidStart string // start time and command line of the current pid, for adoption
	pidCmd   string
	adopted  *child // set before Start to supervise an already running process
}

func newProc(m *Manager, id, instance int, cfg common.AppConfig, env []string) *Proc {
	return &Proc{m: m, id: id, instance: instance, cfg: cfg, env: env,
		status: common.StatusStopped, createdAt: time.Now(), restartReq: make(chan string, 1), reloadReq: make(chan chan error)}
}

func (p *Proc) logBase() string {
	b := common.SafeName(p.cfg.Name)
	if p.cfg.InstancesOrDefault() > 1 {
		b = fmt.Sprintf("%s-%d", b, p.instance)
	}
	return b
}

func (p *Proc) outPath() string {
	if p.cfg.OutFile != "" {
		return p.cfg.OutFile
	}
	return filepath.Join(common.LogsDir(), p.logBase()+"-out.log")
}

func (p *Proc) errPath() string {
	if p.cfg.ErrorFile != "" {
		return p.cfg.ErrorFile
	}
	return filepath.Join(common.LogsDir(), p.logBase()+"-error.log")
}

// Running reports whether the supervise loop is active.
func (p *Proc) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done != nil
}

// Start launches the supervise loop. It is a no-op if the loop already runs.
func (p *Proc) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done != nil {
		return
	}
	p.stopping = false
	p.stopCh = make(chan struct{})
	p.done = make(chan struct{})
	p.status = common.StatusLaunching
	select { // drop a stale soft-restart request from an earlier run
	case <-p.restartReq:
	default:
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	if p.cfg.Watch {
		go p.watchLoop(ctx)
	}
	if p.cfg.CronRestart != "" {
		c := cron.New()
		if _, err := c.AddFunc(p.cfg.CronRestart, func() { p.requestRestart("cron") }); err == nil {
			c.Start()
			go func() { <-ctx.Done(); c.Stop() }()
		}
	}
	go p.loop(p.stopCh, p.done)
}

// requestRestart asks the loop for a restart that does not count as a crash.
func (p *Proc) requestRestart(reason string) {
	select {
	case p.restartReq <- reason:
	default:
	}
}

// Stop signals the child, waits for the kill timeout, then kills the group.
func (p *Proc) Stop() {
	p.mu.Lock()
	if p.done == nil {
		p.mu.Unlock()
		return
	}
	done := p.done
	first := !p.stopping
	if first {
		p.stopping = true
		p.status = common.StatusStopping
		close(p.stopCh)
	}
	pid := p.pid
	timeout := time.Duration(p.cfg.KillTimeoutOrDefault()) * time.Millisecond
	p.mu.Unlock()

	if pid > 0 {
		signalGroup(pid, false)
	}
	select {
	case <-done:
	case <-time.After(timeout):
		if pid > 0 {
			signalGroup(pid, true)
		}
		<-done
	}
}

func (p *Proc) closeLogs() {
	p.mu.Lock()
	ts := []*logTail{p.outT, p.errT}
	fs := []*os.File{p.outF, p.errF}
	p.outT, p.errT, p.outF, p.errF = nil, nil, nil, nil
	p.mu.Unlock()
	for _, f := range fs {
		if f != nil {
			f.Close()
		}
	}
	for _, t := range ts {
		if t != nil {
			t.Close()
		}
	}
}

// drainLogs reports every line written so far; the supervisor calls it when a
// child exits.
func (p *Proc) drainLogs() {
	p.mu.Lock()
	ts := []*logTail{p.outT, p.errT}
	p.mu.Unlock()
	for _, t := range ts {
		if t != nil {
			t.sync()
		}
	}
}

func (p *Proc) logLineHook(stream string) func(string) {
	return func(line string) {
		if h := p.m.logHook(); h != nil {
			h(common.LogLine{TS: time.Now().UnixMilli(), ID: p.id, Instance: p.instance,
				Name: p.cfg.Name, Stream: stream, Line: line})
		}
	}
}

// ensureLogs starts the tailers once and, when writers is set, opens the files
// the child writes to. They stay open across restarts. Caller holds p.mu.
func (p *Proc) ensureLogs(writers bool) error {
	for _, l := range []struct {
		path, stream string
		t            **logTail
		f            **os.File
	}{
		{p.outPath(), "out", &p.outT, &p.outF},
		{p.errPath(), "err", &p.errT, &p.errF},
	} {
		if *l.t == nil {
			t, err := startTail(l.path, p.cfg.LogMaxBytes(), p.cfg.LogRetainOrDefault(), p.logLineHook(l.stream))
			if err != nil {
				return err
			}
			*l.t = t
		}
		if writers && *l.f == nil {
			f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}
			*l.f = f
		}
	}
	return nil
}

// buildCmd prepares the child. Caller holds p.mu.
func (p *Proc) buildCmd() (*exec.Cmd, error) {
	cfg := p.cfg
	var cmd *exec.Cmd
	switch {
	case cfg.Interpreter != "":
		args := append(append(append([]string{}, cfg.InterpreterArgs...), cfg.Script), cfg.Args...)
		cmd = exec.Command(cfg.Interpreter, args...)
	case len(cfg.Args) == 0 && strings.ContainsAny(strings.TrimSpace(cfg.Script), " \t"):
		cmd = exec.Command("sh", "-c", cfg.Script)
	default:
		if i := detectInterpreter(cfg.Script); i != "" {
			cmd = exec.Command(i, append([]string{cfg.Script}, cfg.Args...)...)
		} else {
			cmd = exec.Command(cfg.Script, cfg.Args...)
		}
	}
	cmd.Dir = cfg.Cwd
	env := append([]string{}, p.env...)
	env = append(env,
		fmt.Sprintf("%s=%s", common.EnvName, cfg.Name),
		fmt.Sprintf("%s=%d", common.EnvID, p.id),
		fmt.Sprintf("%s=%d", common.EnvInstance, p.instance),
		fmt.Sprintf("%s=%s", common.EnvNamespace, cfg.Namespace),
	)
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	setProcAttr(cmd)
	if err := p.ensureLogs(true); err != nil {
		return nil, err
	}
	// The child writes straight to the files (no pipe), so it survives the daemon.
	cmd.Stdout, cmd.Stderr = p.outF, p.errF
	return cmd, nil
}

// nextChild returns the adopted process on the first round, a fresh spawn
// afterwards. Caller holds p.mu.
func (p *Proc) nextChild() (*child, error) {
	if a := p.adopted; a != nil {
		p.adopted = nil
		return a, p.ensureLogs(false)
	}
	return p.spawn()
}

func (p *Proc) loop(stopCh, done chan struct{}) {
	defer func() {
		p.mu.Lock()
		p.done = nil
		p.pid = 0
		p.pidStart, p.pidCmd = "", ""
		p.use = usage{}
		cancel := p.cancel
		p.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		close(done)
	}()

	consecutive := 0 // consecutive unstable runs, drives max_restarts and backoff
	reason := "start"
	for {
		p.mu.Lock()
		if p.stopping {
			p.mu.Unlock()
			p.setStatus(common.StatusStopped, "manual")
			return
		}
		c, err := p.nextChild()
		if err != nil {
			p.unstable++
			p.mu.Unlock()
			fmt.Fprintf(os.Stderr, "pm2go: start %q: %v\n", p.cfg.Name, err)
			consecutive++
			if !p.cfg.AutorestartEnabled() || consecutive >= p.cfg.MaxRestartsOrDefault() {
				p.setStatus(common.StatusErrored, "start_failed")
				return
			}
			p.setStatus(common.StatusWaitingRestart, "start_failed")
			if !p.wait(stopCh, p.delay(consecutive)) {
				return
			}
			continue
		}
		p.pid = c.pid
		p.pidStart, p.pidCmd = c.start, c.cmdline
		p.startedAt = c.started
		p.status = common.StatusOnline
		if c.waitReady {
			p.status = common.StatusLaunching // online once the app signals ready
		}
		p.exitCode = nil
		p.mu.Unlock()
		if c.adopted {
			reason = "adopt"
		}
		p.m.persistNow() // a crash right after the spawn must not orphan the child
		if !c.waitReady {
			p.emit("online", reason)
		}

		res := p.supervise(c, stopCh)
		soft, c := res.soft, res.c
		started := c.started
		ran := time.Since(started)
		code := c.exitCode()
		p.drainLogs() // report the last lines before the exit event

		p.mu.Lock()
		p.pid = 0
		p.pidStart, p.pidCmd = "", ""
		p.use = usage{}
		p.exitCode = &code
		stopping := p.stopping
		p.mu.Unlock()
		if stopping {
			p.emit("exit", "manual")
			p.setStatus(common.StatusStopped, "manual")
			return
		}
		if soft != "" {
			p.emit("exit", soft)
			p.mu.Lock()
			p.restarts++
			p.status = common.StatusLaunching
			p.mu.Unlock()
			consecutive = 0
			reason = soft
			continue
		}
		exitReason := "crash"
		if res.timedOut {
			exitReason = "ready_timeout"
		}
		p.emit("exit", exitReason)
		for _, c := range p.cfg.StopExitCodes {
			if c == code {
				p.setStatus(common.StatusStopped, "stop_exit_code")
				return
			}
		}
		// A run that never signalled ready counts as unstable however long it lived.
		neverReady := c.waitReady && !c.isReady
		if neverReady || ran < time.Duration(p.cfg.MinUptimeOrDefault())*time.Millisecond {
			consecutive++
			p.mu.Lock()
			p.unstable++
			p.mu.Unlock()
		} else {
			consecutive = 0
		}
		if !p.cfg.AutorestartEnabled() {
			if code == 0 {
				p.setStatus(common.StatusStopped, "exit")
			} else {
				p.setStatus(common.StatusErrored, "exit")
			}
			return
		}
		if consecutive >= p.cfg.MaxRestartsOrDefault() {
			p.setStatus(common.StatusErrored, "max_restarts")
			return
		}
		p.setStatus(common.StatusWaitingRestart, "crash")
		if !p.wait(stopCh, p.delay(consecutive)) {
			return
		}
		p.mu.Lock()
		p.restarts++
		p.mu.Unlock()
		reason = "crash"
	}
}

// emit reports a state change to the manager's event hook.
func (p *Proc) emit(typ, reason string) {
	p.m.markDirty()
	p.mu.Lock()
	ev := common.Event{
		TS: time.Now().UnixMilli(), ID: p.id, Instance: p.instance, Name: p.cfg.Name,
		Type: typ, Status: p.status, ExitCode: p.exitCode, Reason: reason,
		Manual: p.stopping || reason == "manual",
	}
	p.mu.Unlock()
	if h := p.m.eventHook(); h != nil {
		h(ev)
	}
}

func (p *Proc) setStatus(s, reason string) {
	p.mu.Lock()
	p.status = s
	p.mu.Unlock()
	typ := s
	if s == common.StatusWaitingRestart {
		typ = "waiting"
	}
	p.emit(typ, reason)
}

// delay returns the wait before the next start. n is the consecutive unstable count.
func (p *Proc) delay(n int) time.Duration {
	if base := p.cfg.ExpBackoffRestartDelayMs; base > 0 {
		d := base
		for i := 1; i < n && d < common.MaxExpBackoffDelayMs; i++ {
			d *= 2
		}
		if d > common.MaxExpBackoffDelayMs {
			d = common.MaxExpBackoffDelayMs
		}
		return time.Duration(d) * time.Millisecond
	}
	return time.Duration(p.cfg.RestartDelayMs) * time.Millisecond
}

// wait sleeps for d. It returns false when a stop arrived first, after
// marking the process stopped.
func (p *Proc) wait(stopCh chan struct{}, d time.Duration) bool {
	if d <= 0 {
		select {
		case <-stopCh:
			p.setStatus(common.StatusStopped, "manual")
			return false
		default:
			return true
		}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-stopCh:
		p.setStatus(common.StatusStopped, "manual")
		return false
	}
}

func detectInterpreter(script string) string {
	switch strings.ToLower(filepath.Ext(script)) {
	case ".js", ".mjs", ".cjs":
		return "node"
	case ".py":
		return "python3"
	case ".sh":
		return "sh"
	}
	return ""
}

func (p *Proc) info() common.ProcessInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	var up int64
	if p.status == common.StatusOnline && !p.startedAt.IsZero() {
		up = time.Since(p.startedAt).Milliseconds()
	}
	return common.ProcessInfo{
		ID: p.id, Instance: p.instance, Name: p.cfg.Name, Namespace: p.cfg.Namespace, Status: p.status,
		PID: p.pid, Restarts: p.restarts, Unstable: p.unstable, UptimeMs: up,
		CPU: p.use.cpu, MemoryB: p.use.rss, ExitCode: p.exitCode,
		CreatedAt: p.createdAt.UnixMilli(),
		OutLog:    p.outPath(), ErrLog: p.errPath(), Config: p.cfg,
	}
}
