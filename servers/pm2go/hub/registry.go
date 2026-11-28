package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

const (
	logRingCap   = 5000
	crashKeepFor = 30 * time.Minute
	metricEvery  = 10 * time.Second
	offlineAfter = 15 * time.Second // no snapshot for this long = offline
	commandWait  = 30 * time.Second
	subBuffer    = 512
)

type logEntry struct {
	TS     int64  `json:"ts"`
	Inst   int    `json:"instance"`
	Stream string `json:"stream"`
	Line   string `json:"line"`
}

type ring struct {
	mu   sync.Mutex
	buf  []logEntry
	next int
	full bool
}

func (r *ring) add(e logEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.buf) < logRingCap {
		r.buf = append(r.buf, e)
		return
	}
	r.buf[r.next] = e
	r.next = (r.next + 1) % logRingCap
	r.full = true
}

// last returns up to n newest entries, oldest first.
func (r *ring) last(n int) []logEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make([]logEntry, 0, len(r.buf))
	if r.full {
		all = append(all, r.buf[r.next:]...)
		all = append(all, r.buf[:r.next]...)
	} else {
		all = append(all, r.buf...)
	}
	if n > 0 && len(all) > n {
		all = all[len(all)-n:]
	}
	return all
}

// since returns entries newer than ts (unix ms).
func (r *ring) since(ts int64) []logEntry {
	var out []logEntry
	for _, e := range r.last(0) {
		if e.TS >= ts {
			out = append(out, e)
		}
	}
	return out
}

type agentConn struct {
	name string
	conn *websocket.Conn
	wmu  sync.Mutex

	pmu     sync.Mutex
	pending map[string]chan common.CommandResult
}

func (a *agentConn) write(env common.Envelope) error {
	a.wmu.Lock()
	defer a.wmu.Unlock()
	a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return a.conn.WriteJSON(env)
}

// ServerState is the hub's view of one host.
type ServerState struct {
	Name        string           `json:"name"`
	Hello       common.Hello     `json:"hello"`
	Online      bool             `json:"online"`
	LastSeen    int64            `json:"last_seen"`
	ConnectedAt int64            `json:"connected_at"`
	Snapshot    *common.Snapshot `json:"snapshot,omitempty"`
}

// LogSub receives live log lines for one server/process.
type LogSub struct {
	Server, Proc string // Proc "" = all
	C            chan LogMsg
}

type LogMsg struct {
	Server string `json:"server"`
	Name   string `json:"name"`
	logEntry
}

type Registry struct {
	store *Store

	mu      sync.Mutex
	agents  map[string]*agentConn
	servers map[string]*ServerState
	rings   map[string]*ring // server|name
	crashes map[string][]time.Time
	lastMet map[string]time.Time
	subs    map[*LogSub]struct{}

	Issues *Issues
	seq    uint64
}

func NewRegistry(store *Store, issues *Issues) *Registry {
	r := &Registry{store: store, Issues: issues,
		agents: map[string]*agentConn{}, servers: map[string]*ServerState{},
		rings: map[string]*ring{}, crashes: map[string][]time.Time{},
		lastMet: map[string]time.Time{}, subs: map[*LogSub]struct{}{}}
	r.loadKnown()
	return r
}

func (r *Registry) loadKnown() {
	rows, err := r.store.db.Query(`SELECT name, hello, last_seen FROM servers`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name, hello string
		var seen int64
		if rows.Scan(&name, &hello, &seen) != nil {
			continue
		}
		st := &ServerState{Name: name, LastSeen: seen}
		_ = json.Unmarshal([]byte(hello), &st.Hello)
		r.servers[name] = st
	}
}

func ringKey(server, name string) string { return server + "|" + name }

func (r *Registry) ringFor(server, name string) *ring {
	k := ringKey(server, name)
	r.mu.Lock()
	defer r.mu.Unlock()
	g := r.rings[k]
	if g == nil {
		g = &ring{}
		r.rings[k] = g
	}
	return g
}

// Attach registers a connected agent after its hello. A second connection
// with the same machine name replaces the first.
func (r *Registry) Attach(conn *websocket.Conn, h common.Hello) *agentConn {
	a := &agentConn{name: h.Machine, conn: conn, pending: map[string]chan common.CommandResult{}}
	r.mu.Lock()
	if old := r.agents[a.name]; old != nil {
		old.conn.Close()
	}
	r.agents[a.name] = a
	st := r.servers[a.name]
	if st == nil {
		st = &ServerState{Name: a.name}
		r.servers[a.name] = st
	}
	st.Hello, st.Online = h, true
	st.ConnectedAt, st.LastSeen = time.Now().UnixMilli(), time.Now().UnixMilli()
	r.mu.Unlock()
	b, _ := json.Marshal(h)
	_, _ = r.store.db.Exec(`INSERT INTO servers(name,hello,last_seen) VALUES(?,?,?)
		ON CONFLICT(name) DO UPDATE SET hello=excluded.hello, last_seen=excluded.last_seen`,
		a.name, string(b), time.Now().UnixMilli())
	log.Printf("agent connected: %s (%s)", a.name, h.Hostname)
	return a
}

func (r *Registry) Detach(a *agentConn) {
	r.mu.Lock()
	if r.agents[a.name] == a {
		delete(r.agents, a.name)
		if st := r.servers[a.name]; st != nil {
			st.Online = false
		}
	}
	r.mu.Unlock()
	a.pmu.Lock()
	for id, ch := range a.pending {
		ch <- common.CommandResult{ID: id, Error: "agent disconnected"}
		delete(a.pending, id)
	}
	a.pmu.Unlock()
	log.Printf("agent disconnected: %s", a.name)
}

func (r *Registry) OnSnapshot(server string, s common.Snapshot) {
	now := time.Now()
	r.mu.Lock()
	st := r.servers[server]
	if st == nil {
		r.mu.Unlock()
		return
	}
	st.Snapshot, st.LastSeen, st.Online = &s, now.UnixMilli(), true
	due := now.Sub(r.lastMet[server]) >= metricEvery
	if due {
		r.lastMet[server] = now
	}
	r.mu.Unlock()
	if due {
		r.persistMetrics(server, s)
	}
}

func (r *Registry) persistMetrics(server string, s common.Snapshot) {
	ts := s.TS
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	tx, err := r.store.db.Begin()
	if err != nil {
		return
	}
	sy := s.System
	_, _ = tx.Exec(`INSERT INTO sys_metrics(server,ts,cpu,mem_pct,load1,disk_pct,rx,tx) VALUES(?,?,?,?,?,?,?,?)`,
		server, ts, sy.CPU, sy.MemPercent, sy.Load1, sy.DiskPct, sy.NetRxBps, sy.NetTxBps)
	for _, p := range s.Processes {
		_, _ = tx.Exec(`INSERT INTO proc_metrics(server,name,inst,ts,cpu,mem,restarts,status) VALUES(?,?,?,?,?,?,?,?)`,
			server, p.Name, p.Instance, ts, p.CPU, p.MemoryB, p.Restarts, p.Status)
	}
	if err := tx.Commit(); err != nil {
		log.Printf("persist metrics: %v", err)
	}
	_, _ = r.store.db.Exec(`UPDATE servers SET last_seen=? WHERE name=?`, ts, server)
}

func (r *Registry) OnEvent(server string, e common.Event) {
	manual := 0
	if e.Manual {
		manual = 1
	}
	var code any
	if e.ExitCode != nil {
		code = *e.ExitCode
	}
	_, err := r.store.db.Exec(`INSERT INTO events(ts,server,name,inst,type,status,exit_code,reason,manual) VALUES(?,?,?,?,?,?,?,?,?)`,
		e.TS, server, e.Name, e.Instance, e.Type, e.Status, code, e.Reason, manual)
	if err != nil {
		log.Printf("store event: %v", err)
	}
	// A start that never became ready is as bad as a crash for restart-loop alerts.
	if e.Type == "exit" && !e.Manual && (e.Reason == "crash" || e.Reason == "ready_timeout") {
		k := ringKey(server, e.Name)
		now := time.Now()
		if e.TS > 0 { // replayed events keep their original time
			now = time.UnixMilli(e.TS)
		}
		r.mu.Lock()
		keep := r.crashes[k][:0]
		for _, t := range r.crashes[k] {
			if now.Sub(t) < crashKeepFor {
				keep = append(keep, t)
			}
		}
		r.crashes[k] = append(keep, now)
		r.mu.Unlock()
	}
}

// CrashCount counts unplanned exits of an app within the window.
func (r *Registry) CrashCount(server, name string, window time.Duration) int {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, t := range r.crashes[ringKey(server, name)] {
		if now.Sub(t) <= window {
			n++
		}
	}
	return n
}

func (r *Registry) OnLogs(server string, lines []common.LogLine) {
	if len(lines) == 0 {
		return
	}
	// One lock for the whole batch: snapshot the matching subscribers and the
	// rings once, then work without holding the registry lock.
	r.mu.Lock()
	var subs []*LogSub
	for s := range r.subs {
		if s.Server == server {
			subs = append(subs, s)
		}
	}
	rings := map[string]*ring{}
	for _, l := range lines {
		if _, ok := rings[l.Name]; ok {
			continue
		}
		k := ringKey(server, l.Name)
		g := r.rings[k]
		if g == nil {
			g = &ring{}
			r.rings[k] = g
		}
		rings[l.Name] = g
	}
	r.mu.Unlock()

	for _, l := range lines {
		e := logEntry{TS: l.TS, Inst: l.Instance, Stream: l.Stream, Line: l.Line}
		rings[l.Name].add(e)
		if r.Issues != nil {
			r.Issues.Observe(server, l.Name, l.Line, l.TS)
		}
		for _, s := range subs {
			if s.Proc == "" || s.Proc == l.Name {
				select {
				case s.C <- LogMsg{Server: server, Name: l.Name, logEntry: e}:
				default: // slow viewer: drop
				}
			}
		}
	}
}

func (r *Registry) Subscribe(server, proc string) *LogSub {
	s := &LogSub{Server: server, Proc: proc, C: make(chan LogMsg, subBuffer)}
	r.mu.Lock()
	r.subs[s] = struct{}{}
	r.mu.Unlock()
	return s
}

func (r *Registry) Unsubscribe(s *LogSub) {
	r.mu.Lock()
	delete(r.subs, s)
	r.mu.Unlock()
}

// RecentLogs returns up to n lines of one app, oldest first.
func (r *Registry) RecentLogs(server, name string, n int) []logEntry {
	r.mu.Lock()
	g := r.rings[ringKey(server, name)]
	r.mu.Unlock()
	if g == nil {
		return nil
	}
	return g.last(n)
}

// LogsSince returns lines newer than the time, used by log alert rules.
func (r *Registry) LogsSince(server, name string, since time.Time) []logEntry {
	r.mu.Lock()
	g := r.rings[ringKey(server, name)]
	r.mu.Unlock()
	if g == nil {
		return nil
	}
	return g.since(since.UnixMilli())
}

// States returns a copy of every known server, marking stale ones offline.
func (r *Registry) States() []ServerState {
	now := time.Now().UnixMilli()
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ServerState, 0, len(r.servers))
	for _, s := range r.servers {
		c := *s
		if c.Online && now-c.LastSeen > offlineAfter.Milliseconds() {
			c.Online = false
		}
		out = append(out, c)
	}
	return out
}

func (r *Registry) State(name string) (ServerState, bool) {
	for _, s := range r.States() {
		if s.Name == name {
			return s, true
		}
	}
	return ServerState{}, false
}

// Command sends a command to a connected agent and waits for its result.
func (r *Registry) Command(server string, cmd common.Command) error {
	r.mu.Lock()
	a := r.agents[server]
	r.seq++
	id := fmt.Sprintf("c%d", r.seq)
	r.mu.Unlock()
	if a == nil {
		return errors.New("server is offline")
	}
	cmd.ID = id
	ch := make(chan common.CommandResult, 1)
	a.pmu.Lock()
	a.pending[id] = ch
	a.pmu.Unlock()
	b, _ := json.Marshal(cmd)
	if err := a.write(common.Envelope{Type: common.MsgCommand, Data: b}); err != nil {
		return err
	}
	select {
	case res := <-ch:
		if !res.OK {
			return errors.New(res.Error)
		}
		return nil
	case <-time.After(commandWait):
		a.pmu.Lock()
		delete(a.pending, id)
		a.pmu.Unlock()
		return errors.New("timed out waiting for agent")
	}
}

func (a *agentConn) resolve(res common.CommandResult) {
	a.pmu.Lock()
	ch := a.pending[res.ID]
	delete(a.pending, res.ID)
	a.pmu.Unlock()
	if ch != nil {
		ch <- res
	}
}
