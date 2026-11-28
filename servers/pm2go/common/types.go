package common

import (
	"encoding/json"
	"strings"
)

// Process statuses reported by the daemon.
const (
	StatusLaunching      = "launching"
	StatusOnline         = "online"
	StatusStopping       = "stopping"
	StatusStopped        = "stopped"
	StatusErrored        = "errored"
	StatusWaitingRestart = "waiting restart"
)

// Defaults applied when a field is left empty.
const (
	DefaultMaxRestarts     = 16
	DefaultMinUptimeMs     = 1000
	DefaultKillTimeoutMs   = 1600
	MaxExpBackoffDelayMs   = 15000
	DefaultRestartDelayMs  = 0
	DefaultListenTimeoutMs = 8000
)

// AppConfig is one managed application. It is the unit stored in ecosystem
// files and in the dump file.
type AppConfig struct {
	Name                     string                       `json:"name"`
	Script                   string                       `json:"script"`
	Args                     []string                     `json:"args,omitempty"`
	Cwd                      string                       `json:"cwd,omitempty"`
	Interpreter              string                       `json:"interpreter,omitempty"`
	InterpreterArgs          []string                     `json:"interpreter_args,omitempty"`
	Namespace                string                       `json:"namespace,omitempty"`
	Env                      map[string]string            `json:"env,omitempty"`
	EnvProfiles              map[string]map[string]string `json:"-"`
	Autorestart              *bool                        `json:"autorestart,omitempty"`
	MaxRestarts              int                          `json:"max_restarts,omitempty"`
	MinUptimeMs              int                          `json:"min_uptime,omitempty"`
	RestartDelayMs           int                          `json:"restart_delay,omitempty"`
	ExpBackoffRestartDelayMs int                          `json:"exp_backoff_restart_delay,omitempty"`
	KillTimeoutMs            int                          `json:"kill_timeout,omitempty"`
	StopExitCodes            []int                        `json:"stop_exit_codes,omitempty"`
	OutFile                  string                       `json:"out_file,omitempty"`
	ErrorFile                string                       `json:"error_file,omitempty"`
	Instances                int                          `json:"instances,omitempty"`
	Watch                    bool                         `json:"watch,omitempty"`
	IgnoreWatch              []string                     `json:"ignore_watch,omitempty"`
	WatchDelayMs             int                          `json:"watch_delay,omitempty"`
	CronRestart              string                       `json:"cron_restart,omitempty"`
	MaxMemoryRestartMB       int                          `json:"max_memory_restart,omitempty"`
	LogMaxSizeMB             int                          `json:"log_max_size,omitempty"`
	LogRetain                int                          `json:"log_retain,omitempty"`
	Listen                   string                       `json:"listen,omitempty"`         // shared socket, ":8080" or "unix:/run/app.sock"
	ListenNetwork            string                       `json:"listen_network,omitempty"` // tcp (default), tcp4, tcp6, unix
	WaitReady                bool                         `json:"wait_ready,omitempty"`
	ListenTimeoutMs          int                          `json:"listen_timeout,omitempty"` // max wait for readiness
}

// UnmarshalJSON also collects `env_<profile>` keys into EnvProfiles.
func (c *AppConfig) UnmarshalJSON(data []byte) error {
	type plain AppConfig
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		if !strings.HasPrefix(k, "env_") || len(k) == 4 {
			continue
		}
		var m map[string]string
		if err := json.Unmarshal(v, &m); err != nil {
			return err
		}
		if p.EnvProfiles == nil {
			p.EnvProfiles = map[string]map[string]string{}
		}
		p.EnvProfiles[strings.TrimPrefix(k, "env_")] = m
	}
	*c = AppConfig(p)
	return nil
}

func (c AppConfig) InstancesOrDefault() int {
	if c.Instances < 1 {
		return 1
	}
	return c.Instances
}

func (c AppConfig) LogMaxBytes() int64 {
	if c.LogMaxSizeMB < 0 {
		return 0 // rotation off
	}
	if c.LogMaxSizeMB == 0 {
		return 10 << 20
	}
	return int64(c.LogMaxSizeMB) << 20
}

func (c AppConfig) LogRetainOrDefault() int {
	if c.LogRetain <= 0 {
		return 5
	}
	return c.LogRetain
}

func (c AppConfig) AutorestartEnabled() bool { return c.Autorestart == nil || *c.Autorestart }

func (c AppConfig) MaxRestartsOrDefault() int {
	if c.MaxRestarts <= 0 {
		return DefaultMaxRestarts
	}
	return c.MaxRestarts
}

func (c AppConfig) MinUptimeOrDefault() int {
	if c.MinUptimeMs <= 0 {
		return DefaultMinUptimeMs
	}
	return c.MinUptimeMs
}

func (c AppConfig) KillTimeoutOrDefault() int {
	if c.KillTimeoutMs <= 0 {
		return DefaultKillTimeoutMs
	}
	return c.KillTimeoutMs
}

func (c AppConfig) ListenTimeoutOrDefault() int {
	if c.ListenTimeoutMs <= 0 {
		return DefaultListenTimeoutMs
	}
	return c.ListenTimeoutMs
}

// ListenSpec resolves the shared socket to a (network, address) pair. An
// empty address means no shared socket. "unix:/path" selects a unix socket.
func (c AppConfig) ListenSpec() (network, addr string) {
	if c.Listen == "" {
		return "", ""
	}
	if p, ok := strings.CutPrefix(c.Listen, "unix:"); ok {
		return "unix", p
	}
	if c.ListenNetwork == "" {
		return "tcp", c.Listen
	}
	return c.ListenNetwork, c.Listen
}

// ProcessInfo is the daemon's view of one process, returned by list/describe.
type ProcessInfo struct {
	ID        int       `json:"id"`
	Instance  int       `json:"instance"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Status    string    `json:"status"`
	PID       int       `json:"pid"`
	Restarts  int       `json:"restarts"`
	Unstable  int       `json:"unstable_restarts"`
	UptimeMs  int64     `json:"uptime_ms"`
	CPU       float64   `json:"cpu"`
	MemoryB   int64     `json:"memory"`
	ExitCode  *int      `json:"exit_code,omitempty"`
	CreatedAt int64     `json:"created_at"`
	OutLog    string    `json:"out_log"`
	ErrLog    string    `json:"err_log"`
	Config    AppConfig `json:"config"`
}

// ExportedEnvKeys are variables injected into each child process.
const (
	EnvName      = "PMGO_NAME"
	EnvInstance  = "PMGO_INSTANCE_ID"
	EnvID        = "PMGO_ID"
	EnvHome      = "PMGO_HOME"
	EnvNamespace = "PMGO_NAMESPACE"
	EnvReadyFD   = "PMGO_READY_FD" // write end of the readiness pipe, set with wait_ready
	EnvListen    = "PMGO_LISTEN"   // address of the shared socket, set with listen
)

// Event is a process state change reported to the agent and hub.
type Event struct {
	TS       int64  `json:"ts"` // unix ms
	ID       int    `json:"id"`
	Instance int    `json:"instance"`
	Name     string `json:"name"`
	Type     string `json:"type"` // online, exit, restart, errored, stopped, waiting
	Status   string `json:"status"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Reason   string `json:"reason,omitempty"` // crash, manual, watch, cron, max_memory, ...
	Manual   bool   `json:"manual,omitempty"`
}

// LogLine is one line of child output.
type LogLine struct {
	TS       int64  `json:"ts"`
	ID       int    `json:"id"`
	Instance int    `json:"instance"`
	Name     string `json:"name"`
	Stream   string `json:"stream"` // out or err
	Line     string `json:"line"`
}
