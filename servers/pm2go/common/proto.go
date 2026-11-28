package common

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SystemSample is the host-level view sent with every snapshot.
type SystemSample struct {
	CPU        float64 `json:"cpu"` // percent busy
	MemTotal   int64   `json:"mem_total"`
	MemUsed    int64   `json:"mem_used"`
	MemPercent float64 `json:"mem_percent"`
	Load1      float64 `json:"load1"`
	Load5      float64 `json:"load5"`
	Load15     float64 `json:"load15"`
	DiskTotal  int64   `json:"disk_total"`
	DiskUsed   int64   `json:"disk_used"`
	DiskPct    float64 `json:"disk_percent"`
	NetRxBps   float64 `json:"net_rx_bps"`
	NetTxBps   float64 `json:"net_tx_bps"`
	UptimeS    int64   `json:"uptime_s"`
	CPUCount   int     `json:"cpu_count"`
}

// Agent <-> hub wire protocol: JSON messages over one WebSocket.
const (
	MsgHello    = "hello"    // agent -> hub
	MsgSnapshot = "snapshot" // agent -> hub
	MsgEvent    = "event"    // agent -> hub
	MsgLogs     = "logs"     // agent -> hub
	MsgResult   = "result"   // agent -> hub, reply to MsgCommand
	MsgCommand  = "command"  // hub -> agent
)

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type Hello struct {
	Machine  string `json:"machine"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Version  string `json:"version"`
	CPUCount int    `json:"cpu_count"`
}

type Snapshot struct {
	TS        int64         `json:"ts"`
	System    SystemSample  `json:"system"`
	Processes []ProcessInfo `json:"processes"`
}

type Command struct {
	ID     string     `json:"id"`
	Action string     `json:"action"` // restart, stop, reload, delete, flush, scale, start
	Target string     `json:"target,omitempty"`
	Count  int        `json:"count,omitempty"`
	Config *AppConfig `json:"config,omitempty"`
}

type CommandResult struct {
	ID    string `json:"id"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// LinkConfig tells the daemon's agent where the hub is.
type LinkConfig struct {
	HubURL  string `json:"hub_url"` // e.g. http://hub:8080
	Key     string `json:"key"`
	Machine string `json:"machine"`
}

func LinkFile() string { return filepath.Join(Home(), "link.json") }

func LoadLink() (*LinkConfig, error) {
	b, err := os.ReadFile(LinkFile())
	if err != nil {
		return nil, err
	}
	var l LinkConfig
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

func SaveLink(l LinkConfig) error {
	if err := os.MkdirAll(Home(), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(l, "", "  ")
	return os.WriteFile(LinkFile(), b, 0o600)
}
