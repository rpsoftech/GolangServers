package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// Options tune daemon startup.
type Options struct {
	Resurrect bool
	// OnReady runs after the manager exists, before serving. The agent uses it
	// to install hooks. The returned func is called on link changes.
	OnReady func(*Manager) (onLink func(), onStop func())
}

// Run starts the daemon and blocks until shutdown.
func Run(opts Options) error {
	if err := common.EnsureDirs(); err != nil {
		return err
	}
	sock := common.SocketPath()
	if c, err := net.Dial("unix", sock); err == nil {
		c.Close()
		return errors.New("daemon already running")
	}
	_ = os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return err
	}
	_ = os.Chmod(sock, 0o600)
	_ = os.WriteFile(common.PidFile(), []byte(fmt.Sprint(os.Getpid())), 0o644)

	m := NewManager()
	quit := make(chan struct{})
	s := &server{m: m, quit: quit}
	var onStop func()
	if opts.OnReady != nil {
		s.onLink, onStop = opts.OnReady(m)
	}
	monCtx, monCancel := context.WithCancel(context.Background())
	go m.RunMonitor(monCtx, 2*time.Second)
	// Take over the processes of a previous daemon before anything else
	// starts, so resurrect only adds what is really missing.
	if n := m.Adopt(); n > 0 {
		fmt.Fprintf(os.Stderr, "pm2go: adopted %d running process(es)\n", n)
	}
	m.EnableState(monCtx)
	if opts.Resurrect {
		if _, err := m.Resurrect(); err != nil {
			fmt.Fprintln(os.Stderr, "pm2go: resurrect:", err)
		}
	}

	// SIGINT and SIGTERM mean the daemon goes away (upgrade, restart): the
	// applications keep running and the next daemon adopts them. Only the
	// kill method stops them.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	var killed atomic.Bool
	go func() {
		select {
		case <-sigs:
		case <-quit:
			killed.Store(true)
		}
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			break
		}
		go s.serve(conn)
	}
	monCancel()
	if onStop != nil {
		onStop()
	}
	if killed.Load() {
		m.DisableState() // stop writing first: the next daemon must start empty
		m.StopAll()
	} else {
		m.persistNow() // leave the applications running for the next daemon
	}
	_ = os.Remove(sock)
	_ = os.Remove(common.PidFile())
	return nil
}

type server struct {
	m      *Manager
	quit   chan struct{}
	onLink func()
}

func (s *server) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	enc := json.NewEncoder(conn)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		var req common.Request
		var resp common.Response
		if err := json.Unmarshal(line, &req); err != nil {
			resp = common.Response{Error: "bad request: " + err.Error()}
		} else {
			resp = s.handle(req)
		}
		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}

func ok(v any) common.Response {
	b, err := json.Marshal(v)
	if err != nil {
		return common.Response{Error: err.Error()}
	}
	return common.Response{OK: true, Data: b}
}

func fail(err error) common.Response { return common.Response{Error: err.Error()} }

func (s *server) handle(req common.Request) common.Response {
	var target common.TargetParams
	needTarget := func() error {
		if err := json.Unmarshal(req.Params, &target); err != nil || target.Target == "" {
			return errors.New("target is required")
		}
		return nil
	}
	switch req.Method {
	case common.MethodPing:
		return ok(map[string]int{"pid": os.Getpid()})
	case common.MethodList:
		return ok(s.m.List())
	case common.MethodStart:
		var sp common.StartParams
		if err := json.Unmarshal(req.Params, &sp); err != nil {
			return fail(err)
		}
		infos, err := s.m.Start(sp)
		if err != nil {
			return fail(err)
		}
		return ok(infos)
	case common.MethodScale:
		var sp common.ScaleParams
		if err := json.Unmarshal(req.Params, &sp); err != nil {
			return fail(err)
		}
		infos, err := s.m.Scale(sp.Name, sp.Count)
		if err != nil {
			return fail(err)
		}
		return ok(infos)
	case common.MethodStop, common.MethodRestart, common.MethodReload, common.MethodDelete, common.MethodDescribe:
		if err := needTarget(); err != nil {
			return fail(err)
		}
		var infos []common.ProcessInfo
		var err error
		switch req.Method {
		case common.MethodStop:
			infos, err = s.m.Stop(target.Target)
		case common.MethodRestart:
			infos, err = s.m.Restart(target.Target)
		case common.MethodReload:
			infos, err = s.m.Reload(target.Target)
		case common.MethodDelete:
			infos, err = s.m.Delete(target.Target)
		default:
			infos, err = s.m.Describe(target.Target)
		}
		if err != nil {
			return fail(err)
		}
		return ok(infos)
	case common.MethodFlush:
		if err := needTarget(); err != nil {
			return fail(err)
		}
		if err := s.m.Flush(target.Target); err != nil {
			return fail(err)
		}
		return ok(nil)
	case common.MethodSave:
		n, err := s.m.Save()
		if err != nil {
			return fail(err)
		}
		return ok(map[string]int{"saved": n})
	case common.MethodResurrect:
		infos, err := s.m.Resurrect()
		if err != nil {
			return fail(err)
		}
		return ok(infos)
	case common.MethodLink:
		if s.onLink != nil {
			s.onLink()
		}
		return ok(nil)
	case common.MethodKill:
		go func() { close(s.quit) }()
		return ok(nil)
	}
	return common.Response{Error: "unknown method " + req.Method}
}
