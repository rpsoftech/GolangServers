// Package agent connects the daemon to a hub: it streams snapshots, events
// and log lines out, and runs commands the hub sends back.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
	"github.com/rpsoftech/golang-servers/servers/pm2go/daemon"
)

const (
	snapshotEvery = 2 * time.Second
	logFlushEvery = 500 * time.Millisecond
	logBatchMax   = 200
	writeWait     = 10 * time.Second
	pingEvery     = 20 * time.Second
)

type Agent struct {
	m       *daemon.Manager
	version string

	mu     sync.Mutex
	cancel context.CancelFunc
	out    chan common.Envelope // current connection's queue; nil when offline
	logs   chan common.LogLine
}

func New(m *daemon.Manager, version string) *Agent {
	a := &Agent{m: m, version: version}
	m.SetHooks(a.onEvent, a.onLog)
	return a
}

func (a *Agent) onEvent(e common.Event) { a.send(common.MsgEvent, e) }

func (a *Agent) onLog(l common.LogLine) {
	a.mu.Lock()
	ch := a.logs
	a.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- l:
	default: // drop under backpressure; files still have everything
	}
}

func (a *Agent) send(typ string, v any) {
	a.mu.Lock()
	ch := a.out
	a.mu.Unlock()
	if ch == nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case ch <- common.Envelope{Type: typ, Data: b}:
	default:
	}
}

// Reload re-reads link.json and (re)connects, or disconnects if it is gone.
func (a *Agent) Reload() {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.mu.Unlock()
	link, err := common.LoadLink()
	if err != nil || link.HubURL == "" {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()
	go a.run(ctx, *link)
}

func (a *Agent) Stop() {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.mu.Unlock()
}

func wsURL(hub string) (string, error) {
	u, err := url.Parse(hub)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http", "ws":
		u.Scheme = "ws"
	case "https", "wss":
		u.Scheme = "wss"
	default:
		return "", fmt.Errorf("hub url must start with http:// or https://")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/agent/ws"
	return u.String(), nil
}

func (a *Agent) run(ctx context.Context, link common.LinkConfig) {
	backoff := time.Second
	for ctx.Err() == nil {
		start := time.Now()
		err := a.session(ctx, link)
		if ctx.Err() != nil {
			return
		}
		fmt.Fprintf(os.Stderr, "pm2go agent: %v; retrying in %s\n", err, backoff)
		if time.Since(start) > time.Minute {
			backoff = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (a *Agent) session(ctx context.Context, link common.LinkConfig) error {
	u, err := wsURL(link.HubURL)
	if err != nil {
		return err
	}
	d := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	hdr := map[string][]string{"Authorization": {"Bearer " + link.Key}}
	conn, _, err := d.DialContext(ctx, u, hdr)
	if err != nil {
		return err
	}
	defer conn.Close()

	out := make(chan common.Envelope, 4096)
	logs := make(chan common.LogLine, 8192)
	a.mu.Lock()
	a.out, a.logs = out, logs
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.out, a.logs = nil, nil
		a.mu.Unlock()
	}()

	host, _ := os.Hostname()
	machine := link.Machine
	if machine == "" {
		machine = host
	}
	a.send(common.MsgHello, common.Hello{Machine: machine, Hostname: host, OS: runtime.GOOS,
		Version: a.version, CPUCount: runtime.NumCPU()})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 3)

	// writer: the only goroutine that writes to conn
	go func() {
		ping := time.NewTicker(pingEvery)
		defer ping.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case env := <-out:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteJSON(env); err != nil {
					errCh <- err
					return
				}
			case <-ping.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	// producer: snapshots and log batches
	go func() {
		snap := time.NewTicker(snapshotEvery)
		flush := time.NewTicker(logFlushEvery)
		defer snap.Stop()
		defer flush.Stop()
		var batch []common.LogLine
		emit := func() {
			if len(batch) > 0 {
				a.send(common.MsgLogs, batch)
				batch = nil
			}
		}
		a.sendSnapshot()
		for {
			select {
			case <-ctx.Done():
				return
			case <-snap.C:
				a.sendSnapshot()
			case l := <-logs:
				batch = append(batch, l)
				if len(batch) >= logBatchMax {
					emit()
				}
			case <-flush.C:
				emit()
			}
		}
	}()

	// reader: commands from the hub
	go func() {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			return nil
		})
		for {
			var env common.Envelope
			if err := conn.ReadJSON(&env); err != nil {
				errCh <- err
				return
			}
			conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			if env.Type == common.MsgCommand {
				var cmd common.Command
				if json.Unmarshal(env.Data, &cmd) == nil {
					go func() { a.send(common.MsgResult, a.exec(cmd)) }()
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (a *Agent) sendSnapshot() {
	a.send(common.MsgSnapshot, common.Snapshot{
		TS: time.Now().UnixMilli(), System: a.m.System(), Processes: a.m.List(),
	})
}

// exec runs one hub command against the local manager.
func (a *Agent) exec(c common.Command) common.CommandResult {
	res := common.CommandResult{ID: c.ID, OK: true}
	var err error
	switch c.Action {
	case "restart":
		_, err = a.m.Restart(c.Target)
	case "reload":
		_, err = a.m.Reload(c.Target)
	case "stop":
		_, err = a.m.Stop(c.Target)
	case "delete":
		_, err = a.m.Delete(c.Target)
	case "flush":
		err = a.m.Flush(c.Target)
	case "scale":
		_, err = a.m.Scale(c.Target, c.Count)
	case "start":
		if c.Config == nil {
			err = errors.New("config required")
		} else {
			_, err = a.m.Start(common.StartParams{Config: *c.Config})
		}
	case "save":
		_, err = a.m.Save()
	default:
		err = fmt.Errorf("unknown action %q", c.Action)
	}
	if err != nil {
		res.OK, res.Error = false, err.Error()
	}
	return res
}
