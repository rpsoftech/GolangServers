package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// Manager owns the process table.
type Manager struct {
	mu     sync.Mutex
	procs  []*Proc
	nextID int
	lns    *listeners

	hookMu sync.RWMutex
	onEv   func(common.Event)
	onLog  func(common.LogLine)

	sampler *Sampler
	sysMu   sync.Mutex
	sys     SystemSample
}

func NewManager() *Manager { return &Manager{sampler: NewSampler(), lns: newListeners()} }

// SetHooks registers receivers for process events and log lines. They run on
// supervisor goroutines and must not block.
func (m *Manager) SetHooks(ev func(common.Event), lg func(common.LogLine)) {
	m.hookMu.Lock()
	m.onEv, m.onLog = ev, lg
	m.hookMu.Unlock()
}

func (m *Manager) eventHook() func(common.Event) {
	m.hookMu.RLock()
	defer m.hookMu.RUnlock()
	return m.onEv
}

func (m *Manager) logHook() func(common.LogLine) {
	m.hookMu.RLock()
	defer m.hookMu.RUnlock()
	return m.onLog
}

// byName returns all instances with the name, ordered by instance.
func (m *Manager) byName(name string) []*Proc {
	var out []*Proc
	for _, p := range m.procs {
		if p.cfg.Name == name {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].instance < out[j].instance })
	return out
}

// resolve returns the processes matching a target: "all", numeric id, name or namespace.
func (m *Manager) resolve(target string) ([]*Proc, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if target == "all" {
		return append([]*Proc{}, m.procs...), nil
	}
	if id, err := strconv.Atoi(target); err == nil {
		for _, p := range m.procs {
			if p.id == id {
				return []*Proc{p}, nil
			}
		}
	}
	var out []*Proc
	for _, p := range m.procs {
		if p.cfg.Name == target || (p.cfg.Namespace != "" && p.cfg.Namespace == target) {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("process or namespace %q not found", target)
	}
	return out, nil
}

func validate(cfg *common.AppConfig) error {
	if cfg.Script == "" {
		return fmt.Errorf("script is required")
	}
	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}
	if cfg.Listen != "" {
		switch network, _ := cfg.ListenSpec(); network {
		case "tcp", "tcp4", "tcp6", "unix":
		default:
			return fmt.Errorf("listen_network %q is not supported (tcp, tcp4, tcp6, unix)", network)
		}
	}
	if cfg.CronRestart != "" {
		if _, err := cron.ParseStandard(cfg.CronRestart); err != nil {
			return fmt.Errorf("cron_restart: %w", err)
		}
	}
	return nil
}

func (m *Manager) addProc(cfg common.AppConfig, instance int, env []string) *Proc {
	p := newProc(m, m.nextID, instance, cfg, env)
	m.nextID++
	m.procs = append(m.procs, p)
	return p
}

// Start adds an app, or updates the config of an existing one and restarts it.
// With instances > 1 it keeps exactly that many processes under the name.
func (m *Manager) Start(sp common.StartParams) ([]common.ProcessInfo, error) {
	cfg := sp.Config
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	env := sp.CLIEnv
	if len(env) == 0 {
		env = os.Environ()
	}
	n := cfg.InstancesOrDefault()
	if cfg.Listen != "" {
		if _, err := m.lns.acquire(cfg); err != nil { // report bind errors to the caller
			return nil, err
		}
	} else {
		m.lns.release(cfg.Name)
	}

	m.mu.Lock()
	existing := m.byName(cfg.Name)
	var started []*Proc
	var toRestart, extras []*Proc
	for i := 0; i < n; i++ {
		var p *Proc
		for _, e := range existing {
			if e.instance == i {
				p = e
			}
		}
		if p == nil {
			p = m.addProc(cfg, i, env)
			started = append(started, p)
		} else {
			toRestart = append(toRestart, p)
		}
	}
	for _, e := range existing {
		if e.instance >= n {
			extras = append(extras, e)
		}
	}
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, p := range toRestart {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Stop()
			p.closeLogs()
			p.mu.Lock()
			p.cfg, p.env = cfg, env
			p.mu.Unlock()
			p.Start()
		}()
	}
	for _, p := range extras {
		wg.Add(1)
		go func() { defer wg.Done(); m.remove(p) }()
	}
	wg.Wait()
	for _, p := range started {
		p.Start()
	}
	all := append(started, toRestart...)
	return m.infos(all), nil
}

func (m *Manager) remove(p *Proc) {
	p.Stop()
	p.closeLogs()
	m.mu.Lock()
	kept := m.procs[:0]
	for _, q := range m.procs {
		if q != p {
			kept = append(kept, q)
		}
	}
	m.procs = kept
	last := len(m.byName(p.cfg.Name)) == 0
	m.mu.Unlock()
	if last {
		m.lns.release(p.cfg.Name) // delete or scale to zero closes the shared socket
	}
}

func (m *Manager) Stop(target string) ([]common.ProcessInfo, error) {
	ps, err := m.resolve(target)
	if err != nil {
		return nil, err
	}
	parallel(ps, func(p *Proc) { p.Stop() })
	return m.infos(ps), nil
}

func (m *Manager) Restart(target string) ([]common.ProcessInfo, error) {
	ps, err := m.resolve(target)
	if err != nil {
		return nil, err
	}
	parallel(ps, func(p *Proc) {
		p.Stop()
		p.mu.Lock()
		p.restarts++
		p.mu.Unlock()
		p.Start()
	})
	return m.infos(ps), nil
}

// Reload restarts processes one at a time per app name so multi-instance apps
// keep serving. Apps with a shared socket get an overlapping reload: the new
// process starts and signals ready before the old one gets SIGTERM, so the
// socket never goes unserved. Other apps are stopped, started and awaited.
func (m *Manager) Reload(target string) ([]common.ProcessInfo, error) {
	ps, err := m.resolve(target)
	if err != nil {
		return nil, err
	}
	groups := map[string][]*Proc{}
	for _, p := range ps {
		groups[p.cfg.Name] = append(groups[p.cfg.Name], p)
	}
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var errs []error
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool { return g[i].instance < g[j].instance })
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, p := range g {
				if err := reloadProc(p); err != nil {
					errMu.Lock()
					errs = append(errs, fmt.Errorf("%s[%d]: %w", p.cfg.Name, p.instance, err))
					errMu.Unlock()
					return // keep the remaining instances on the old code
				}
			}
		}()
	}
	wg.Wait()
	return m.infos(ps), errors.Join(errs...)
}

func reloadProc(p *Proc) error {
	p.mu.Lock()
	overlap := p.cfg.Listen != ""
	p.mu.Unlock()
	if overlap {
		if handled, err := p.Reload(); handled {
			return err
		}
	}
	p.Stop()
	p.mu.Lock()
	p.restarts++
	min := time.Duration(p.cfg.MinUptimeOrDefault()) * time.Millisecond
	limit := 15 * time.Second
	if p.cfg.WaitReady {
		min = 0 // online already means ready
		limit = time.Duration(p.cfg.ListenTimeoutOrDefault())*time.Millisecond + 5*time.Second
	}
	p.mu.Unlock()
	p.Start()
	waitStable(p, min, limit)
	p.mu.Lock()
	status := p.status
	p.mu.Unlock()
	if status != common.StatusOnline {
		return fmt.Errorf("did not come back online after reload (status %q)", status)
	}
	return nil
}

func waitStable(p *Proc, stable, limit time.Duration) {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		ok := p.status == common.StatusOnline && time.Since(p.startedAt) >= stable
		gone := p.done == nil
		p.mu.Unlock()
		if ok || gone {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Scale changes the instance count of an app without restarting existing ones.
func (m *Manager) Scale(name string, count int) ([]common.ProcessInfo, error) {
	if count < 0 {
		return nil, fmt.Errorf("count must be >= 0")
	}
	m.mu.Lock()
	existing := m.byName(name)
	if len(existing) == 0 {
		m.mu.Unlock()
		return nil, fmt.Errorf("process %q not found", name)
	}
	cfg := existing[0].cfg
	env := existing[0].env
	var add []*Proc
	var drop []*Proc
	for i := len(existing); i < count; i++ {
		c := cfg
		c.Instances = count
		add = append(add, m.addProc(c, i, env))
	}
	for _, e := range existing {
		if e.instance >= count {
			drop = append(drop, e)
		}
	}
	for _, e := range existing {
		e.mu.Lock()
		e.cfg.Instances = count
		e.mu.Unlock()
	}
	m.mu.Unlock()
	parallel(drop, m.remove)
	for _, p := range add {
		p.Start()
	}
	m.mu.Lock()
	cur := m.byName(name)
	m.mu.Unlock()
	return m.infos(cur), nil
}

func (m *Manager) Delete(target string) ([]common.ProcessInfo, error) {
	ps, err := m.resolve(target)
	if err != nil {
		return nil, err
	}
	infos := m.infos(ps)
	parallel(ps, m.remove)
	return infos, nil
}

func parallel(ps []*Proc, f func(*Proc)) {
	var wg sync.WaitGroup
	for _, p := range ps {
		wg.Add(1)
		go func() { defer wg.Done(); f(p) }()
	}
	wg.Wait()
}

func (m *Manager) infos(ps []*Proc) []common.ProcessInfo {
	out := make([]common.ProcessInfo, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.info())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Manager) List() []common.ProcessInfo {
	m.mu.Lock()
	ps := append([]*Proc{}, m.procs...)
	m.mu.Unlock()
	return m.infos(ps)
}

func (m *Manager) Describe(target string) ([]common.ProcessInfo, error) {
	ps, err := m.resolve(target)
	if err != nil {
		return nil, err
	}
	return m.infos(ps), nil
}

// System returns the latest host sample.
func (m *Manager) System() SystemSample {
	m.sysMu.Lock()
	defer m.sysMu.Unlock()
	return m.sys
}

// RunMonitor samples host and process usage until ctx ends, and enforces
// max_memory_restart.
func (m *Manager) RunMonitor(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	m.sample()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.sample()
		}
	}
}

func (m *Manager) sample() {
	m.mu.Lock()
	ps := append([]*Proc{}, m.procs...)
	m.mu.Unlock()
	var pids []int
	for _, p := range ps {
		p.mu.Lock()
		if p.pid > 0 {
			pids = append(pids, p.pid)
		}
		p.mu.Unlock()
	}
	us := m.sampler.SampleProcs(pids)
	for _, p := range ps {
		p.mu.Lock()
		if u, ok := us[p.pid]; ok && p.pid > 0 {
			p.use = u
		}
		limit := int64(p.cfg.MaxMemoryRestartMB) << 20
		over := limit > 0 && p.pid > 0 && p.use.rss > limit
		p.mu.Unlock()
		if over {
			p.requestRestart("max_memory")
		}
	}
	s := m.sampler.SampleSystem()
	m.sysMu.Lock()
	m.sys = s
	m.sysMu.Unlock()
}

// Flush truncates the log files of the targets.
func (m *Manager) Flush(target string) error {
	ps, err := m.resolve(target)
	if err != nil {
		return err
	}
	for _, p := range ps {
		p.mu.Lock()
		ws := []*RotatingFile{p.outW, p.errW}
		paths := []string{p.outPath(), p.errPath()}
		p.mu.Unlock()
		for i, w := range ws {
			if w != nil {
				if err := w.Truncate(); err != nil {
					return err
				}
			} else if err := os.Truncate(paths[i], 0); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

// Save writes every configured app to the dump file (one entry per name).
func (m *Manager) Save() (int, error) {
	m.mu.Lock()
	seen := map[string]bool{}
	cfgs := make([]common.AppConfig, 0, len(m.procs))
	for _, p := range m.procs {
		if seen[p.cfg.Name] {
			continue
		}
		seen[p.cfg.Name] = true
		cfgs = append(cfgs, p.cfg)
	}
	m.mu.Unlock()
	b, err := json.MarshalIndent(cfgs, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(common.DumpFile()), 0o755); err != nil {
		return 0, err
	}
	tmp := common.DumpFile() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return 0, err
	}
	return len(cfgs), os.Rename(tmp, common.DumpFile())
}

// Resurrect starts every app from the dump file that is not already known.
func (m *Manager) Resurrect() ([]common.ProcessInfo, error) {
	b, err := os.ReadFile(common.DumpFile())
	if err != nil {
		return nil, fmt.Errorf("read dump: %w (run save first)", err)
	}
	var cfgs []common.AppConfig
	if err := json.Unmarshal(b, &cfgs); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}
	var out []common.ProcessInfo
	for _, c := range cfgs {
		m.mu.Lock()
		exists := len(m.byName(c.Name)) > 0
		m.mu.Unlock()
		if exists {
			continue
		}
		infos, err := m.Start(common.StartParams{Config: c})
		if err != nil {
			return out, err
		}
		out = append(out, infos...)
	}
	return out, nil
}

// StopAll stops every process; used on daemon shutdown.
func (m *Manager) StopAll() {
	_, _ = m.Stop("all")
	m.lns.closeAll()
}
