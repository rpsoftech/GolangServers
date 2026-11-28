package hub

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

//go:embed ui
var uiFS embed.FS

type API struct {
	cfg    Config
	auth   *auth
	store  *Store
	reg    *Registry
	engine *Engine
	notif  *Notifier
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	ui, _ := fs.Sub(uiFS, "ui")
	mux.Handle("GET /", http.FileServer(http.FS(ui)))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /agent/ws", a.agentWS)
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) { a.auth.logout(w); w.WriteHeader(204) })
	mux.HandleFunc("GET /api/me", a.auth.require(func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]bool{"ok": true}) }))

	req := a.auth.require
	mux.HandleFunc("GET /api/servers", req(a.servers))
	mux.HandleFunc("GET /api/servers/{name}", req(a.server))
	mux.HandleFunc("GET /api/servers/{name}/metrics", req(a.metrics))
	mux.HandleFunc("GET /api/servers/{name}/logs", req(a.logs))
	mux.HandleFunc("POST /api/servers/{name}/command", req(a.command))
	mux.HandleFunc("GET /api/logs/ws", req(a.logsWS))
	mux.HandleFunc("GET /api/events", req(a.events))
	mux.HandleFunc("GET /api/issues", req(a.issues))
	mux.HandleFunc("POST /api/issues/status", req(a.issueStatus))

	mux.HandleFunc("GET /api/alerts/rules", req(func(w http.ResponseWriter, r *http.Request) { writeJSON(w, a.engine.Rules()) }))
	mux.HandleFunc("POST /api/alerts/rules", req(a.addRule))
	mux.HandleFunc("PUT /api/alerts/rules/{id}", req(a.updateRule))
	mux.HandleFunc("DELETE /api/alerts/rules/{id}", req(a.deleteRule))
	mux.HandleFunc("GET /api/alerts/active", req(func(w http.ResponseWriter, r *http.Request) { writeJSON(w, a.engine.Active()) }))
	mux.HandleFunc("GET /api/alerts/history", req(a.history))

	mux.HandleFunc("GET /api/settings/info", req(a.info))
	mux.HandleFunc("GET /api/settings/telegram", req(a.getTelegram))
	mux.HandleFunc("PUT /api/settings/telegram", req(a.putTelegram))
	mux.HandleFunc("POST /api/settings/telegram/test", req(a.testTelegram))
	return secureHeaders(mux)
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data:")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func httpErr(w http.ResponseWriter, code int, msg string) { http.Error(w, msg, code) }

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		httpErr(w, 400, "bad json: "+err.Error())
		return false
	}
	return true
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &b) {
		return
	}
	if a.auth.login(w, r, b.Password) {
		writeJSON(w, map[string]bool{"ok": true})
	}
}

// ---- agents ----

var agentUp = websocket.Upgrader{ReadBufferSize: 1 << 16, WriteBufferSize: 1 << 16,
	CheckOrigin: func(*http.Request) bool { return true }} // agents are not browsers; auth is the key

func (a *API) agentWS(w http.ResponseWriter, r *http.Request) {
	if !a.auth.agentKeyOK(r) {
		httpErr(w, 401, "bad agent key")
		return
	}
	conn, err := agentUp.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(8 << 20)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	var env common.Envelope
	if err := conn.ReadJSON(&env); err != nil || env.Type != common.MsgHello {
		return
	}
	var h common.Hello
	if json.Unmarshal(env.Data, &h) != nil || h.Machine == "" || len(h.Machine) > 100 {
		return
	}
	ag := a.reg.Attach(conn, h)
	defer a.reg.Detach(ag)
	ping := func(string) error { conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil }
	conn.SetPingHandler(func(s string) error {
		_ = ping(s)
		ag.wmu.Lock()
		defer ag.wmu.Unlock()
		return conn.WriteControl(websocket.PongMessage, []byte(s), time.Now().Add(5*time.Second))
	})
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	for {
		var env common.Envelope
		if err := conn.ReadJSON(&env); err != nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		switch env.Type {
		case common.MsgSnapshot:
			var s common.Snapshot
			if json.Unmarshal(env.Data, &s) == nil {
				a.reg.OnSnapshot(ag.name, s)
			}
		case common.MsgEvent:
			var e common.Event
			if json.Unmarshal(env.Data, &e) == nil {
				a.reg.OnEvent(ag.name, e)
			}
		case common.MsgLogs:
			var l []common.LogLine
			if json.Unmarshal(env.Data, &l) == nil {
				a.reg.OnLogs(ag.name, l)
			}
		case common.MsgResult:
			var res common.CommandResult
			if json.Unmarshal(env.Data, &res) == nil {
				ag.resolve(res)
			}
		}
	}
}

// ---- servers ----

type serverSummary struct {
	Name      string               `json:"name"`
	Online    bool                 `json:"online"`
	Hostname  string               `json:"hostname"`
	LastSeen  int64                `json:"last_seen"`
	System    *common.SystemSample `json:"system,omitempty"`
	Total     int                  `json:"processes"`
	Running   int                  `json:"online_processes"`
	Errored   int                  `json:"errored_processes"`
	AlertsMax string               `json:"alert_level,omitempty"`
}

func (a *API) servers(w http.ResponseWriter, r *http.Request) {
	states := a.reg.States()
	worst := map[string]string{}
	for _, al := range a.engine.Active() {
		if al.State == "firing" && levelRank[al.Level] > levelRank[worst[al.Server]] {
			worst[al.Server] = al.Level
		}
	}
	out := make([]serverSummary, 0, len(states))
	for _, s := range states {
		sum := serverSummary{Name: s.Name, Online: s.Online, Hostname: s.Hello.Hostname, LastSeen: s.LastSeen, AlertsMax: worst[s.Name]}
		if s.Snapshot != nil {
			sy := s.Snapshot.System
			sum.System = &sy
			sum.Total = len(s.Snapshot.Processes)
			for _, p := range s.Snapshot.Processes {
				switch p.Status {
				case common.StatusOnline:
					sum.Running++
				case common.StatusErrored:
					sum.Errored++
				}
			}
		}
		out = append(out, sum)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, out)
}

func (a *API) server(w http.ResponseWriter, r *http.Request) {
	s, ok := a.reg.State(r.PathValue("name"))
	if !ok {
		httpErr(w, 404, "unknown server")
		return
	}
	writeJSON(w, s)
}

var ranges = map[string]time.Duration{"15m": 15 * time.Minute, "1h": time.Hour, "6h": 6 * time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour}

func (a *API) metrics(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	d, ok := ranges[r.URL.Query().Get("range")]
	if !ok {
		d = time.Hour
	}
	from := time.Now().Add(-d).UnixMilli()
	step := max(d.Milliseconds()/240, 10_000)
	type sysPt struct {
		TS      int64   `json:"ts"`
		CPU     float64 `json:"cpu"`
		MemPct  float64 `json:"mem_pct"`
		Load1   float64 `json:"load1"`
		DiskPct float64 `json:"disk_pct"`
		Rx      float64 `json:"rx"`
		Tx      float64 `json:"tx"`
	}
	type procPt struct {
		TS       int64   `json:"ts"`
		CPU      float64 `json:"cpu"`
		Mem      float64 `json:"mem"`
		Restarts int64   `json:"restarts"`
	}
	resp := struct {
		System []sysPt             `json:"system"`
		Procs  map[string][]procPt `json:"procs"`
	}{System: []sysPt{}, Procs: map[string][]procPt{}}

	rows, err := a.store.db.Query(`SELECT (ts/?)*? b, AVG(cpu), AVG(mem_pct), AVG(load1), AVG(disk_pct), AVG(rx), AVG(tx)
		FROM sys_metrics WHERE server=? AND ts>=? GROUP BY b ORDER BY b`, step, step, name, from)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	for rows.Next() {
		var p sysPt
		if rows.Scan(&p.TS, &p.CPU, &p.MemPct, &p.Load1, &p.DiskPct, &p.Rx, &p.Tx) == nil {
			resp.System = append(resp.System, p)
		}
	}
	rows.Close()
	rows, err = a.store.db.Query(`SELECT name, (ts/?)*? b, AVG(c), AVG(m), MAX(r) FROM
		(SELECT name, ts, SUM(cpu) c, SUM(mem) m, SUM(restarts) r FROM proc_metrics WHERE server=? AND ts>=? GROUP BY name, ts)
		GROUP BY name, b ORDER BY b`, step, step, name, from)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	for rows.Next() {
		var n string
		var p procPt
		if rows.Scan(&n, &p.TS, &p.CPU, &p.Mem, &p.Restarts) == nil {
			resp.Procs[n] = append(resp.Procs[n], p)
		}
	}
	rows.Close()
	writeJSON(w, resp)
}

func (a *API) logs(w http.ResponseWriter, r *http.Request) {
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	if n <= 0 || n > 2000 {
		n = 200
	}
	l := a.reg.RecentLogs(r.PathValue("name"), r.URL.Query().Get("proc"), n)
	if l == nil {
		l = []logEntry{}
	}
	writeJSON(w, l)
}

var allowedActions = map[string]bool{"restart": true, "reload": true, "stop": true, "delete": true, "flush": true, "scale": true, "start": true, "save": true}

func (a *API) command(w http.ResponseWriter, r *http.Request) {
	var c common.Command
	if !readJSON(w, r, &c) {
		return
	}
	if !allowedActions[c.Action] {
		httpErr(w, 400, "unknown action")
		return
	}
	if c.Action != "save" && c.Action != "start" && c.Target == "" {
		httpErr(w, 400, "target required")
		return
	}
	server := r.PathValue("name")
	log.Printf("command from %s: server=%s action=%s target=%s", clientIP(r), server, c.Action, c.Target)
	if err := a.reg.Command(server, c); err != nil {
		httpErr(w, 502, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

var browserUp = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		o := r.Header.Get("Origin")
		if o == "" {
			return true
		}
		u, err := url.Parse(o)
		return err == nil && u.Host == r.Host
	},
}

func (a *API) logsWS(w http.ResponseWriter, r *http.Request) {
	server, proc := r.URL.Query().Get("server"), r.URL.Query().Get("proc")
	if server == "" {
		httpErr(w, 400, "server required")
		return
	}
	conn, err := browserUp.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	sub := a.reg.Subscribe(server, proc)
	defer a.reg.Unsubscribe(sub)
	if proc != "" {
		for _, e := range a.reg.RecentLogs(server, proc, 200) {
			if conn.WriteJSON(LogMsg{Server: server, Name: proc, logEntry: e}) != nil {
				return
			}
		}
	}
	gone := make(chan struct{})
	go func() { // detect client close
		defer close(gone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-gone:
			return
		case m := <-sub.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if conn.WriteJSON(m) != nil {
				return
			}
		case <-ping.C:
			if conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)) != nil {
				return
			}
		}
	}
}

func (a *API) events(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `SELECT ts,server,name,inst,type,status,exit_code,reason,manual FROM events`
	args := []any{}
	if s := r.URL.Query().Get("server"); s != "" {
		q += ` WHERE server=?`
		args = append(args, s)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := a.store.db.Query(q, args...)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var ts, inst, manual int64
		var server, name, typ, status, reason string
		var code *int64
		if rows.Scan(&ts, &server, &name, &inst, &typ, &status, &code, &reason, &manual) != nil {
			continue
		}
		out = append(out, map[string]any{"ts": ts, "server": server, "name": name, "instance": inst, "type": typ,
			"status": status, "exit_code": code, "reason": reason, "manual": manual == 1})
	}
	writeJSON(w, out)
}

func (a *API) issues(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.reg.Issues.List(r.URL.Query().Get("status")))
}

func (a *API) issueStatus(w http.ResponseWriter, r *http.Request) {
	var b struct{ Server, Name, Fingerprint, Status string }
	if !readJSON(w, r, &b) {
		return
	}
	if b.Status != "open" && b.Status != "resolved" && b.Status != "ignored" {
		httpErr(w, 400, "bad status")
		return
	}
	if !a.reg.Issues.SetStatus(b.Server, b.Name, b.Fingerprint, b.Status) {
		httpErr(w, 404, "issue not found")
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// ---- alerts ----

func ruleID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpErr(w, 400, "bad id")
		return 0, false
	}
	return id, true
}

func (a *API) addRule(w http.ResponseWriter, r *http.Request) {
	var rule Rule
	if !readJSON(w, r, &rule) {
		return
	}
	out, err := a.engine.AddRule(rule)
	if err != nil {
		httpErr(w, 400, err.Error())
		return
	}
	writeJSON(w, out)
}

func (a *API) updateRule(w http.ResponseWriter, r *http.Request) {
	id, ok := ruleID(w, r)
	if !ok {
		return
	}
	var rule Rule
	if !readJSON(w, r, &rule) {
		return
	}
	out, err := a.engine.UpdateRule(id, rule)
	if err != nil {
		httpErr(w, 400, err.Error())
		return
	}
	writeJSON(w, out)
}

func (a *API) deleteRule(w http.ResponseWriter, r *http.Request) {
	id, ok := ruleID(w, r)
	if !ok {
		return
	}
	if err := a.engine.DeleteRule(id); err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	w.WriteHeader(204)
}

func (a *API) history(w http.ResponseWriter, r *http.Request) {
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if n <= 0 || n > 500 {
		n = 100
	}
	h, err := a.engine.History(n)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	writeJSON(w, h)
}

// ---- settings ----

func (a *API) info(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"agent_key": a.cfg.AgentKey, "retention_days": a.cfg.RetentionDays})
}

type telegramView struct {
	Enabled  bool           `json:"enabled"`
	TokenSet bool           `json:"token_set"`
	Chats    []TelegramChat `json:"chats"`
}

func (a *API) getTelegram(w http.ResponseWriter, r *http.Request) {
	s := a.notif.Settings()
	if s.Chats == nil {
		s.Chats = []TelegramChat{}
	}
	writeJSON(w, telegramView{Enabled: s.Enabled, TokenSet: s.BotToken != "", Chats: s.Chats})
}

func (a *API) putTelegram(w http.ResponseWriter, r *http.Request) {
	var s TelegramSettings
	if !readJSON(w, r, &s) {
		return
	}
	if err := a.notif.Update(s); err != nil {
		httpErr(w, 400, err.Error())
		return
	}
	a.getTelegram(w, r)
}

func (a *API) testTelegram(w http.ResponseWriter, r *http.Request) {
	var b struct {
		ChatID string `json:"chat_id"`
	}
	if !readJSON(w, r, &b) {
		return
	}
	if b.ChatID == "" {
		httpErr(w, 400, "chat_id required")
		return
	}
	if err := a.notif.SendTest(b.ChatID, FormatAlert("firing", "info", "Test message from pm2go", "hub", "", "If you can read this, Telegram alerts work.")); err != nil {
		httpErr(w, 502, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}
