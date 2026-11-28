package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

const stateVersion = 1

// stateFile is $PMGO_HOME/state.json: what a new daemon needs to take over the
// processes of a dead one.
type stateFile struct {
	Version   int          `json:"version"`
	SavedAt   int64        `json:"saved_at"`
	DaemonPID int          `json:"daemon_pid"`
	NextID    int          `json:"next_id"`
	Procs     []procRecord `json:"procs"`
}

// procRecord is one process. PID, StartTime and Cmdline identify the OS
// process; the rest restores the supervisor state.
type procRecord struct {
	ID        int              `json:"id"`
	Instance  int              `json:"instance"`
	Config    common.AppConfig `json:"config"`
	Env       []string         `json:"env,omitempty"`
	PID       int              `json:"pid"`
	StartTime string           `json:"start_time,omitempty"`
	Cmdline   string           `json:"cmdline,omitempty"`
	Restarts  int              `json:"restarts"`
	Unstable  int              `json:"unstable"`
	CreatedAt int64            `json:"created_at"`
	StartedAt int64            `json:"started_at,omitempty"`
	Status    string           `json:"status"`
	ExitCode  *int             `json:"exit_code,omitempty"`
}

// EnableState turns on state persistence: a coalescing writer goroutine runs
// until ctx ends. Call it after Adopt, so the old file is read before it is
// replaced.
func (m *Manager) EnableState(ctx context.Context) {
	if !m.stateOn.CompareAndSwap(false, true) {
		return
	}
	m.persistNow()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-m.dirty:
				select { // let a burst of changes settle into one write
				case <-time.After(50 * time.Millisecond):
				case <-ctx.Done():
					return
				}
				m.persistNow()
			}
		}
	}()
}

// markDirty asks the writer for a state write. It never blocks.
func (m *Manager) markDirty() {
	select {
	case m.dirty <- struct{}{}:
	default:
	}
}

// persistNow writes the state file now. It is a no-op until EnableState.
func (m *Manager) persistNow() {
	if !m.stateOn.Load() {
		return
	}
	m.persistMu.Lock()
	defer m.persistMu.Unlock()
	if !m.stateOn.Load() {
		return
	}
	if err := writeFileAtomic(common.StateFile(), m.snapshot()); err != nil {
		fmt.Fprintln(os.Stderr, "pm2go: write state:", err)
	}
}

// DisableState stops writing and removes the state file; `pm2go kill` uses it
// so the next daemon starts empty.
func (m *Manager) DisableState() {
	m.persistMu.Lock()
	defer m.persistMu.Unlock()
	m.stateOn.Store(false)
	_ = os.Remove(common.StateFile())
}

func (m *Manager) snapshot() []byte {
	m.mu.Lock()
	ps := append([]*Proc{}, m.procs...)
	next := m.nextID
	m.mu.Unlock()
	sf := stateFile{Version: stateVersion, SavedAt: time.Now().UnixMilli(), DaemonPID: os.Getpid(), NextID: next}
	for _, p := range ps {
		sf.Procs = append(sf.Procs, p.record())
	}
	b, _ := json.MarshalIndent(sf, "", "  ")
	return b
}

func (p *Proc) record() procRecord {
	p.mu.Lock()
	defer p.mu.Unlock()
	r := procRecord{
		ID: p.id, Instance: p.instance, Config: p.cfg, Env: p.env, PID: p.pid,
		StartTime: p.pidStart, Cmdline: p.pidCmd, Restarts: p.restarts, Unstable: p.unstable,
		CreatedAt: p.createdAt.UnixMilli(), Status: p.status, ExitCode: p.exitCode,
	}
	if !p.startedAt.IsZero() {
		r.StartedAt = p.startedAt.UnixMilli()
	}
	return r
}

// writeFileAtomic replaces path with data: a temp file in the same directory,
// fsync, rename. A reader sees the old or the new file, never a partial one.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	name := f.Name()
	cleanup := func(err error) error { f.Close(); os.Remove(name); return err }
	if err := f.Chmod(0o600); err != nil {
		return cleanup(err)
	}
	if _, err := f.Write(data); err != nil {
		return cleanup(err)
	}
	if err := f.Sync(); err != nil {
		return cleanup(err)
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	if d, err := os.Open(dir); err == nil { // make the rename durable
		_ = d.Sync()
		d.Close()
	}
	return nil
}
