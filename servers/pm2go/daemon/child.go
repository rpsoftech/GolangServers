package daemon

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// child is one running OS process of a Proc: the command, its exit channel and
// the readiness state. A Proc normally has one child; during an overlapping
// reload it briefly has two.
type child struct {
	cmd     *exec.Cmd
	pid     int
	waitCh  chan error // receives once when the process exits
	started time.Time

	waitReady bool
	ready     chan struct{} // closed when the app signalled ready; nil without wait_ready
	rdPipe    *os.File
	isReady   bool // set by the supervise loop after ready fired
	rdOnce    sync.Once
}

func (c *child) closeReadPipe() {
	if c.rdPipe != nil {
		c.rdOnce.Do(func() { _ = c.rdPipe.Close() })
	}
}

// spawn starts a new OS process for the proc. Caller holds p.mu. The shared
// listener, when configured, is descriptor 3; the readiness pipe follows it.
func (p *Proc) spawn() (*child, error) {
	cmd, err := p.buildCmd()
	if err != nil {
		return nil, err
	}
	var files []*os.File
	if p.cfg.Listen != "" {
		sl, err := p.m.lns.acquire(p.cfg)
		if err != nil {
			return nil, err
		}
		files = append(files, sl.file)
		cmd.Env = append(cmd.Env,
			"LISTEN_FDS=1",
			"LISTEN_FDNAMES="+common.SafeName(p.cfg.Name),
			common.EnvListen+"="+p.cfg.Listen,
		)
	}
	c := &child{cmd: cmd, waitReady: p.cfg.WaitReady}
	var wr *os.File
	if c.waitReady {
		var rd *os.File
		if rd, wr, err = os.Pipe(); err != nil {
			return nil, err
		}
		c.rdPipe = rd
		files = append(files, wr)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", common.EnvReadyFD, 2+len(files)))
	}
	cmd.ExtraFiles = files
	err = cmd.Start()
	if wr != nil {
		_ = wr.Close() // the child holds its own copy
	}
	if err != nil {
		c.closeReadPipe()
		return nil, err
	}
	c.pid = cmd.Process.Pid
	c.started = time.Now()
	c.waitCh = make(chan error, 1)
	go func() { c.waitCh <- cmd.Wait() }()
	if c.waitReady {
		c.ready = make(chan struct{})
		go readReady(c.rdPipe, c.ready)
	}
	return c, nil
}

// readReady closes ready when the pipe delivers a "ready" line.
func readReady(r *os.File, ready chan struct{}) {
	line, _ := bufio.NewReader(r).ReadString('\n')
	if strings.TrimSpace(line) == "ready" { // tolerate \r\n and trailing blanks from non-Go apps
		close(ready)
	}
}

// terminate sends SIGTERM to the child's group and escalates to SIGKILL after
// the kill timeout. It returns once the process is gone.
func (p *Proc) terminate(c *child) {
	p.mu.Lock()
	timeout := time.Duration(p.cfg.KillTimeoutOrDefault()) * time.Millisecond
	p.mu.Unlock()
	signalGroup(c.pid, false)
	select {
	case <-c.waitCh:
	case <-time.After(timeout):
		signalGroup(c.pid, true)
		<-c.waitCh
	}
	c.closeReadPipe()
}

// markReady switches a launching child to online.
func (p *Proc) markReady(c *child) {
	c.isReady = true
	p.mu.Lock()
	if p.pid == c.pid && p.status == common.StatusLaunching {
		p.status = common.StatusOnline
		p.startedAt = time.Now()
	}
	p.mu.Unlock()
	p.emit("online", "ready")
}

// superviseResult tells loop how the current child ended.
type superviseResult struct {
	c        *child // the child that exited (differs from the first after a reload)
	soft     string // soft restart reason, if one ended it
	timedOut bool   // wait_ready window expired
}

// supervise blocks until the child exits, a soft restart or stop arrives, or
// the readiness window expires. It also serves overlapping reloads.
func (p *Proc) supervise(c *child, stopCh chan struct{}) superviseResult {
	var timeout <-chan time.Time
	if c.ready != nil {
		t := time.NewTimer(time.Duration(p.cfg.ListenTimeoutOrDefault()) * time.Millisecond)
		defer t.Stop()
		timeout = t.C
	}
	readyCh := c.ready
	for {
		select {
		case <-c.waitCh:
			c.closeReadPipe()
			return superviseResult{c: c}
		case <-readyCh:
			p.markReady(c)
			c.closeReadPipe() // the readiness line was read; free the descriptor now
			readyCh, timeout = nil, nil
		case <-timeout:
			fmt.Fprintf(os.Stderr, "pm2go: %q not ready after %dms, killing\n", p.cfg.Name, p.cfg.ListenTimeoutOrDefault())
			p.terminate(c)
			return superviseResult{c: c, timedOut: true}
		case soft := <-p.restartReq:
			p.terminate(c)
			return superviseResult{c: c, soft: soft}
		case <-stopCh:
			<-c.waitCh // Stop escalates to SIGKILL itself
			c.closeReadPipe()
			return superviseResult{c: c}
		case rq := <-p.reloadReq:
			nc, err := p.overlap(c, stopCh)
			rq <- err
			if nc != nil {
				c, readyCh, timeout = nc, nil, nil
			}
		}
	}
}

var errReloadStopped = errors.New("stopped during reload")

// overlap starts a replacement child next to the running one, waits until it
// is ready (or alive for min_uptime without wait_ready), then swaps and drains
// the old one with SIGTERM. On failure the old child keeps running.
func (p *Proc) overlap(old *child, stopCh chan struct{}) (*child, error) {
	p.mu.Lock()
	nc, err := p.spawn()
	cfg := p.cfg
	p.mu.Unlock()
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*child, error) {
		p.terminate(nc)
		return nil, err
	}
	if nc.waitReady {
		t := time.NewTimer(time.Duration(cfg.ListenTimeoutOrDefault()) * time.Millisecond)
		defer t.Stop()
		select {
		case <-nc.ready:
		case <-t.C:
			return fail(fmt.Errorf("instance %d not ready after %dms", p.instance, cfg.ListenTimeoutOrDefault()))
		case <-nc.waitCh:
			nc.closeReadPipe()
			return nil, fmt.Errorf("instance %d exited before it was ready", p.instance)
		case <-stopCh:
			return fail(errReloadStopped)
		}
	} else {
		t := time.NewTimer(time.Duration(cfg.MinUptimeOrDefault()) * time.Millisecond)
		defer t.Stop()
		select {
		case <-t.C:
		case <-nc.waitCh:
			return nil, fmt.Errorf("instance %d exited during reload", p.instance)
		case <-stopCh:
			return fail(errReloadStopped)
		}
	}
	nc.isReady = true
	nc.closeReadPipe() // no more readiness reads once the replacement is live
	p.mu.Lock()
	p.pid = nc.pid
	p.startedAt = time.Now()
	p.status = common.StatusOnline
	p.exitCode = nil
	p.restarts++
	p.use = usage{}
	p.mu.Unlock()
	p.emit("online", "reload")
	p.terminate(old)
	return nc, nil
}

// Reload replaces the running child without a gap. The bool is false when the
// proc has no live supervise loop to serve the request; the caller then falls
// back to stop and start.
func (p *Proc) Reload() (bool, error) {
	p.mu.Lock()
	done := p.done
	stopping := p.stopping
	p.mu.Unlock()
	if done == nil || stopping {
		return false, nil
	}
	rq := make(chan error, 1)
	select {
	case p.reloadReq <- rq:
	case <-done:
		return false, nil
	case <-time.After(time.Second):
		return false, nil // loop is busy with a restart delay
	}
	select {
	case err := <-rq:
		return true, err
	case <-done:
		return true, errors.New("process stopped during reload")
	}
}
