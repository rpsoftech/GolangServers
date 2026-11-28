package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// Adopt takes over the processes recorded in the state file by a previous
// daemon. Entries whose process is still the same one are supervised again
// without a restart. Entries whose process is gone (or whose pid now belongs
// to another program) follow the restart policy. It returns the number of
// adopted processes.
func (m *Manager) Adopt() int {
	path := common.StateFile()
	b, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "pm2go: read state:", err)
		}
		return 0
	}
	var sf stateFile
	if err := json.Unmarshal(b, &sf); err != nil || sf.Version != stateVersion {
		if err == nil {
			err = fmt.Errorf("unsupported version %d", sf.Version)
		}
		fmt.Fprintf(os.Stderr, "pm2go: state file unusable (%v), starting fresh; kept as %s.corrupt\n", err, path)
		_ = os.Rename(path, path+".corrupt")
		return 0
	}
	adopted := 0
	m.mu.Lock()
	if sf.NextID > m.nextID {
		m.nextID = sf.NextID
	}
	m.mu.Unlock()
	for _, r := range sf.Procs {
		if m.adoptRecord(r) {
			adopted++
		}
	}
	return adopted
}

// verifyRecord returns "" when the recorded pid is still the recorded process.
func verifyRecord(r procRecord) string {
	if r.StartTime == "" {
		return "no start time recorded"
	}
	st, err := procStartTime(r.PID)
	if err != nil || !processAlive(r.PID) {
		return "not running"
	}
	if st != r.StartTime {
		return "pid reused by another process (start time differs)"
	}
	if cl, err := procCmdline(r.PID); err != nil || cl != r.Cmdline {
		return "command line differs"
	}
	return ""
}

func (m *Manager) adoptRecord(r procRecord) bool {
	p := newProc(m, r.ID, r.Instance, r.Config, r.Env)
	p.restarts, p.unstable, p.exitCode = r.Restarts, r.Unstable, r.ExitCode
	if r.CreatedAt > 0 {
		p.createdAt = time.UnixMilli(r.CreatedAt)
	}
	m.mu.Lock()
	m.procs = append(m.procs, p)
	if r.ID >= m.nextID {
		m.nextID = r.ID + 1
	}
	m.mu.Unlock()

	switch {
	case r.PID > 0:
		why := verifyRecord(r)
		if why == "" {
			p.mu.Lock()
			p.adopted = newAdoptedChild(r)
			p.mu.Unlock()
			m.lns.adopt(r.Config, r.PID)
			p.Start()
			return true
		}
		fmt.Fprintf(os.Stderr, "pm2go: %s[%d] pid %d not adopted: %s\n", r.Config.Name, r.Instance, r.PID, why)
		// The process died while the daemon was down. Never signal that pid.
		p.mu.Lock()
		code := -1
		p.exitCode = &code
		p.mu.Unlock()
		p.emit("exit", "daemon_down")
		if r.Config.AutorestartEnabled() {
			p.mu.Lock()
			p.restarts++
			p.mu.Unlock()
			p.Start()
		} else {
			p.setStatus(common.StatusErrored, "exit")
		}
	case r.Status == common.StatusStopped || r.Status == common.StatusErrored || r.Status == common.StatusStopping:
		p.status = r.Status
		if r.Status == common.StatusStopping {
			p.status = common.StatusStopped
		}
	default: // waiting for a restart or launching when the daemon died
		p.Start()
	}
	return false
}

// newAdoptedChild wraps a running process of a previous daemon. Its exit is
// observed with a pidfd on Linux and by polling elsewhere; the exit code is
// unknown.
func newAdoptedChild(r procRecord) *child {
	c := &child{pid: r.PID, start: r.StartTime, cmdline: r.Cmdline, adopted: true,
		waitCh: make(chan error, 1), started: time.Now()}
	if r.StartedAt > 0 {
		c.started = time.UnixMilli(r.StartedAt)
	}
	exited := watchExit(r.PID, r.StartTime)
	go func() { <-exited; c.waitCh <- nil }()
	return c
}
