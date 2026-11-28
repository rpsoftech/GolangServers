package hub

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

type fakeTelegram struct {
	srv  *httptest.Server
	mu   sync.Mutex
	msgs []map[string]any
}

func newFakeTelegram(t *testing.T) *fakeTelegram {
	f := &fakeTelegram{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		m["path"] = r.URL.Path
		f.mu.Lock()
		f.msgs = append(f.msgs, m)
		f.mu.Unlock()
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeTelegram) wait(t *testing.T, n int) []map[string]any {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		if len(f.msgs) >= n {
			out := append([]map[string]any{}, f.msgs...)
			f.mu.Unlock()
			return out
		}
		f.mu.Unlock()
		time.Sleep(25 * time.Millisecond)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	t.Fatalf("got %d telegram messages, want %d: %v", len(f.msgs), n, f.msgs)
	return nil
}

func (f *fakeTelegram) count() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.msgs) }

type env struct {
	store *Store
	reg   *Registry
	notif *Notifier
	eng   *Engine
	tg    *fakeTelegram
	clock time.Time
}

func setup(t *testing.T) *env {
	t.Helper()
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	reg := NewRegistry(store, NewIssues(store))
	n := NewNotifier(store)
	tg := newFakeTelegram(t)
	n.apiBase = tg.srv.URL
	if err := n.Update(TelegramSettings{Enabled: true, BotToken: "TOKEN", Chats: []TelegramChat{
		{Name: "all", ChatID: "1", MinLevel: "info"}, {Name: "oncall", ChatID: "2", MinLevel: "critical"}}}); err != nil {
		t.Fatal(err)
	}
	eng, err := NewEngine(store, reg, n)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range eng.Rules() { // start from a clean rule set
		_ = eng.DeleteRule(r.ID)
	}
	e := &env{store: store, reg: reg, notif: n, eng: eng, tg: tg, clock: time.Now()}
	eng.now = func() time.Time { return e.clock }
	return e
}

func (e *env) setSnapshot(server string, cpu float64, procs ...common.ProcessInfo) {
	e.reg.mu.Lock()
	defer e.reg.mu.Unlock()
	e.reg.servers[server] = &ServerState{Name: server, Online: true, LastSeen: time.Now().UnixMilli(),
		Snapshot: &common.Snapshot{System: common.SystemSample{CPU: cpu, MemPercent: 10, CPUCount: 4}, Processes: procs}}
}

func proc(name string, cpu float64, status string) common.ProcessInfo {
	return common.ProcessInfo{Name: name, CPU: cpu, Status: status}
}

func (e *env) advance(d time.Duration) { e.clock = e.clock.Add(d); e.eng.Evaluate() }

func TestSustainedCPUFiresAndResolves(t *testing.T) {
	e := setup(t)
	if _, err := e.eng.AddRule(Rule{Name: "cpu", Enabled: true, Level: "warning", Type: TypeProcCPU, Threshold: 80, DurationS: 60, NotifyResolved: true}); err != nil {
		t.Fatal(err)
	}
	e.setSnapshot("s1", 5, proc("api", 90, "online"))
	e.eng.Evaluate()
	if a := e.eng.Active(); len(a) != 1 || a[0].State != "pending" {
		t.Fatalf("want one pending alert, got %+v", a)
	}
	e.advance(30 * time.Second)
	if e.tg.count() != 0 {
		t.Fatal("notified before duration elapsed")
	}
	e.advance(31 * time.Second)
	if a := e.eng.Active(); a[0].State != "firing" {
		t.Fatalf("want firing, got %s", a[0].State)
	}
	msgs := e.tg.wait(t, 1)
	if msgs[0]["chat_id"] != "1" || !strings.Contains(msgs[0]["text"].(string), "WARNING") {
		t.Fatalf("bad message: %v", msgs[0])
	}
	// warning must not reach the critical-only chat
	time.Sleep(1300 * time.Millisecond)
	if e.tg.count() != 1 {
		t.Fatalf("warning reached %d chats, want 1", e.tg.count())
	}
	e.setSnapshot("s1", 5, proc("api", 10, "online"))
	e.advance(time.Second) // starts the recovery timer
	e.advance(11 * time.Second)
	if len(e.eng.Active()) != 0 {
		t.Fatalf("alert should be resolved: %+v", e.eng.Active())
	}
	msgs = e.tg.wait(t, 2)
	if !strings.Contains(msgs[1]["text"].(string), "RESOLVED") {
		t.Fatalf("no resolved message: %v", msgs[1])
	}
}

func TestBriefSpikeDoesNotFire(t *testing.T) {
	e := setup(t)
	e.eng.AddRule(Rule{Name: "cpu", Enabled: true, Level: "warning", Type: TypeProcCPU, Threshold: 80, DurationS: 60})
	e.setSnapshot("s1", 5, proc("api", 90, "online"))
	e.eng.Evaluate()
	e.advance(20 * time.Second)
	e.setSnapshot("s1", 5, proc("api", 10, "online"))
	e.advance(time.Second)
	e.advance(60 * time.Second)
	if len(e.eng.Active()) != 0 || e.tg.count() != 0 {
		t.Fatal("short spike must not alert")
	}
}

func TestHigherLevelCoversLower(t *testing.T) {
	e := setup(t)
	e.eng.AddRule(Rule{Name: "warn", Enabled: true, Level: "warning", Type: TypeSystemCPU, Threshold: 80})
	e.eng.AddRule(Rule{Name: "crit", Enabled: true, Level: "critical", Type: TypeSystemCPU, Threshold: 95})
	e.setSnapshot("s1", 97)
	e.eng.Evaluate()
	e.advance(time.Second)
	msgs := e.tg.wait(t, 2) // critical goes to both chats; the covered warning is silent
	for _, m := range msgs {
		if !strings.Contains(m["text"].(string), "CRITICAL") {
			t.Fatalf("unexpected message %v", m["text"])
		}
	}
	time.Sleep(1300 * time.Millisecond)
	if e.tg.count() != 2 {
		t.Fatalf("got %d messages, want 2", e.tg.count())
	}
	covered := 0
	for _, a := range e.eng.Active() {
		if a.Suppress {
			covered++
		}
	}
	if covered != 1 {
		t.Fatalf("covered alerts = %d, want 1", covered)
	}
	// critical clears, warning still true: it must now notify
	e.setSnapshot("s1", 85)
	e.advance(time.Second)
	e.advance(11 * time.Second)
	e.tg.wait(t, 3)
}

func TestRepeatReminders(t *testing.T) {
	e := setup(t)
	e.eng.AddRule(Rule{Name: "mem", Enabled: true, Level: "critical", Type: TypeSystemMemory, Threshold: 50, RepeatS: 600, Chats: []string{"2"}})
	e.reg.mu.Lock()
	e.reg.servers["s1"] = &ServerState{Name: "s1", Online: true, LastSeen: time.Now().UnixMilli(),
		Snapshot: &common.Snapshot{System: common.SystemSample{MemPercent: 90, MemTotal: 8 << 30}}}
	e.reg.mu.Unlock()
	e.eng.Evaluate()
	e.advance(time.Second)
	e.tg.wait(t, 1)
	e.advance(590 * time.Second)
	time.Sleep(1300 * time.Millisecond)
	if e.tg.count() != 1 {
		t.Fatal("reminder sent too early")
	}
	e.advance(15 * time.Second)
	msgs := e.tg.wait(t, 2)
	if !strings.Contains(msgs[1]["text"].(string), "still firing") {
		t.Fatalf("not a reminder: %v", msgs[1]["text"])
	}
}

func TestLogErrorRateAndRestartLoop(t *testing.T) {
	e := setup(t)
	e.eng.AddRule(Rule{Name: "errs", Enabled: true, Level: "warning", Type: TypeLogErrors, Count: 3, WindowS: 60, Chats: []string{"1"}})
	e.eng.AddRule(Rule{Name: "loop", Enabled: true, Level: "critical", Type: TypeRestartLoop, Count: 2, WindowS: 300, Chats: []string{"1"}})
	e.setSnapshot("s1", 5, proc("api", 1, "online"))
	now := time.Now().UnixMilli()
	e.reg.OnLogs("s1", []common.LogLine{
		{TS: now, Name: "api", Stream: "err", Line: "ERROR db timeout 1"}, {TS: now, Name: "api", Stream: "out", Line: "all fine"},
		{TS: now, Name: "api", Stream: "err", Line: "panic: nil map"}, {TS: now, Name: "api", Stream: "err", Line: "fatal: boom"}})
	for i := 0; i < 2; i++ {
		e.reg.OnEvent("s1", common.Event{TS: now, Name: "api", Type: "exit", Reason: "crash"})
	}
	e.eng.Evaluate()
	e.advance(time.Second)
	msgs := e.tg.wait(t, 2)
	var texts []string
	for _, m := range msgs {
		texts = append(texts, m["text"].(string))
	}
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, "errs") || !strings.Contains(joined, "loop") || !strings.Contains(joined, "3 matching") {
		t.Fatalf("messages: %s", joined)
	}
}

func TestServerOfflineAndErroredProcess(t *testing.T) {
	e := setup(t)
	e.eng.AddRule(Rule{Name: "down", Enabled: true, Level: "critical", Type: TypeServerOffl, DurationS: 30, Chats: []string{"1"}})
	e.eng.AddRule(Rule{Name: "errored", Enabled: true, Level: "critical", Type: TypeProcStatus, Chats: []string{"1"}})
	e.setSnapshot("s1", 5, proc("api", 0, "errored"))
	e.eng.Evaluate()
	e.advance(time.Second)
	e.tg.wait(t, 1)
	e.reg.mu.Lock()
	e.reg.servers["s1"].Online = false
	e.reg.servers["s1"].LastSeen = e.clock.UnixMilli()
	e.reg.mu.Unlock()
	e.advance(5 * time.Second)
	e.advance(40 * time.Second)
	msgs := e.tg.wait(t, 2)
	if !strings.Contains(msgs[1]["text"].(string), "down") {
		t.Fatalf("want offline alert, got %v", msgs[1]["text"])
	}
}

func TestRuleValidation(t *testing.T) {
	bad := []Rule{
		{Name: "", Level: "info", Type: TypeSystemCPU, Threshold: 1},
		{Name: "x", Level: "loud", Type: TypeSystemCPU, Threshold: 1},
		{Name: "x", Level: "info", Type: "nope"},
		{Name: "x", Level: "info", Type: TypeSystemCPU},
		{Name: "x", Level: "info", Type: TypeLogErrors, Count: 1, WindowS: 1, Pattern: "("},
		{Name: "x", Level: "info", Type: TypeRestartLoop},
	}
	for i, r := range bad {
		if r.Validate() == nil {
			t.Errorf("rule %d should be invalid", i)
		}
	}
}

func TestDefaultRulesSeeded(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	eng, err := NewEngine(store, NewRegistry(store, nil), NewNotifier(store))
	if err != nil {
		t.Fatal(err)
	}
	levels := map[string]bool{}
	for _, r := range eng.Rules() {
		levels[r.Level] = true
	}
	if len(eng.Rules()) < 10 || !levels["warning"] || !levels["critical"] {
		t.Fatalf("defaults = %d rules, levels %v", len(eng.Rules()), levels)
	}
}

func TestFingerprintGroupsSimilarErrors(t *testing.T) {
	a := Fingerprint(`2026-01-02T03:04:05Z ERROR user 123 not found at 0xdeadbeef12 "abc"`)
	b := Fingerprint(`2026-02-03T10:00:00Z ERROR user 98765 not found at 0xcafebabe99 "xyz"`)
	c := Fingerprint(`2026-02-03T10:00:00Z ERROR connection refused`)
	if a != b || a == c {
		t.Fatalf("fingerprints a=%s b=%s c=%s", a, b, c)
	}
	if IsErrorLine("all good") || !IsErrorLine("Traceback (most recent call last)") {
		t.Fatal("classifier wrong")
	}
}

func TestTelegramUpdateKeepsTokenAndValidates(t *testing.T) {
	e := setup(t)
	if err := e.notif.Update(TelegramSettings{Enabled: true, Chats: []TelegramChat{{ChatID: "9"}}}); err != nil {
		t.Fatal(err)
	}
	s := e.notif.Settings()
	if s.BotToken != "TOKEN" || s.Chats[0].MinLevel != "warning" {
		t.Fatalf("settings = %+v", s)
	}
	if e.notif.Update(TelegramSettings{Chats: []TelegramChat{{ChatID: "9", MinLevel: "loud"}}}) == nil {
		t.Fatal("bad level accepted")
	}
	if e.notif.Update(TelegramSettings{Chats: []TelegramChat{{Name: "x"}}}) == nil {
		t.Fatal("empty chat id accepted")
	}
}

// ---- API + agent protocol ----

func newAPI(t *testing.T) (*httptest.Server, *API, Config) {
	e := setup(t)
	cfg := Config{AdminPassword: "pw", AgentKey: "key", SessionSecret: "secret", RetentionDays: 1}
	api := &API{cfg: cfg, auth: newAuth(cfg), store: e.store, reg: e.reg, engine: e.eng, notif: e.notif}
	srv := httptest.NewServer(api.Handler())
	t.Cleanup(srv.Close)
	return srv, api, cfg
}

func TestAPIAuthAndAgentFlow(t *testing.T) {
	srv, _, _ := newAPI(t)
	if r, err := http.Get(srv.URL + "/api/servers"); err != nil || r.StatusCode != 401 {
		t.Fatalf("unauthenticated request: err=%v response=%v", err, r)
	}
	if r, err := http.Get(srv.URL + "/"); err != nil || r.StatusCode != 200 {
		t.Fatalf("ui request: err=%v response=%v", err, r)
	}
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/agent/ws"
	if _, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Authorization": {"Bearer wrong"}}); err == nil || resp.StatusCode != 401 {
		t.Fatalf("bad agent key accepted: %v", err)
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Authorization": {"Bearer key"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	send := func(typ string, v any) {
		b, _ := json.Marshal(v)
		if err := conn.WriteJSON(common.Envelope{Type: typ, Data: b}); err != nil {
			t.Fatal(err)
		}
	}
	send(common.MsgHello, common.Hello{Machine: "box1", Hostname: "h1", OS: "linux", CPUCount: 2})
	send(common.MsgSnapshot, common.Snapshot{TS: time.Now().UnixMilli(), System: common.SystemSample{CPU: 12, MemPercent: 30, DiskPct: 40},
		Processes: []common.ProcessInfo{{ID: 0, Name: "api", Status: "online", CPU: 3, MemoryB: 1 << 20}}})
	send(common.MsgLogs, []common.LogLine{{TS: time.Now().UnixMilli(), Name: "api", Stream: "err", Line: "ERROR it broke"}})
	agentGot := make(chan common.Command, 1)
	go func() {
		for {
			var env common.Envelope
			if conn.ReadJSON(&env) != nil {
				return
			}
			var c common.Command
			if env.Type == common.MsgCommand && json.Unmarshal(env.Data, &c) == nil {
				agentGot <- c
				send(common.MsgResult, common.CommandResult{ID: c.ID, OK: true})
			}
		}
	}()

	jar, _ := cookiejar.New(nil)
	cl := &http.Client{Jar: jar}
	post := func(path, body string) *http.Response {
		r, err := cl.Post(srv.URL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if r := post("/api/login", `{"password":"nope"}`); r.StatusCode != 401 {
		t.Fatalf("bad login = %d", r.StatusCode)
	}
	time.Sleep(2100 * time.Millisecond) // lockout window after a failure
	if r := post("/api/login", `{"password":"pw"}`); r.StatusCode != 200 {
		t.Fatalf("login = %d", r.StatusCode)
	}
	deadline := time.Now().Add(3 * time.Second)
	var list []serverSummary
	for time.Now().Before(deadline) {
		r := mustGet(t, cl, srv.URL+"/api/servers")
		list = nil
		_ = json.NewDecoder(r.Body).Decode(&list)
		if len(list) == 1 && list[0].Total == 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(list) != 1 || list[0].Name != "box1" || !list[0].Online || list[0].Running != 1 {
		t.Fatalf("servers = %+v", list)
	}
	r := mustGet(t, cl, srv.URL+"/api/issues")
	var issues []Issue
	_ = json.NewDecoder(r.Body).Decode(&issues)
	if len(issues) != 1 || issues[0].Name != "api" {
		t.Fatalf("issues = %+v", issues)
	}
	r = mustGet(t, cl, srv.URL+"/api/servers/box1/logs?proc=api")
	var logs []logEntry
	_ = json.NewDecoder(r.Body).Decode(&logs)
	if len(logs) != 1 {
		t.Fatalf("logs = %+v", logs)
	}
	if r := post("/api/servers/box1/command", `{"action":"restart","target":"api"}`); r.StatusCode != 200 {
		b, _ := io.ReadAll(r.Body)
		t.Fatalf("command = %d %s", r.StatusCode, b)
	}
	if c := <-agentGot; c.Action != "restart" || c.Target != "api" {
		t.Fatalf("agent got %+v", c)
	}
	if r := post("/api/servers/box1/command", `{"action":"rm -rf","target":"x"}`); r.StatusCode != 400 {
		t.Fatalf("bad action = %d", r.StatusCode)
	}
	if r := post("/api/servers/ghost/command", `{"action":"restart","target":"x"}`); r.StatusCode != 502 {
		t.Fatalf("offline server = %d", r.StatusCode)
	}
	// telegram settings never leak the token
	r = mustGet(t, cl, srv.URL+"/api/settings/telegram")
	b, _ := io.ReadAll(r.Body)
	if strings.Contains(string(b), "TOKEN") || !strings.Contains(string(b), `"token_set":true`) {
		t.Fatalf("telegram view leaks token or hides state: %s", b)
	}
	// metrics endpoint answers with series arrays
	r = mustGet(t, cl, srv.URL+"/api/servers/box1/metrics?range=1h")
	if r.StatusCode != 200 {
		t.Fatalf("metrics = %d", r.StatusCode)
	}
}

func TestLogsWebsocketStreamsLive(t *testing.T) {
	srv, api, _ := newAPI(t)
	req, _ := http.NewRequest("GET", srv.URL+"/api/logs/ws?server=s1&proc=api", nil)
	_ = req
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/api/logs/ws?server=s1&proc=api",
		http.Header{"Authorization": {"Bearer pw"}})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	time.Sleep(100 * time.Millisecond)
	api.reg.OnLogs("s1", []common.LogLine{{TS: 1, Name: "api", Stream: "out", Line: "hello live"}})
	ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	var m LogMsg
	if err := ws.ReadJSON(&m); err != nil || m.Line != "hello live" {
		t.Fatalf("got %+v, %v", m, err)
	}
}

func TestTrustProxyClientIPAndSecureCookie(t *testing.T) {
	defer func(old bool) { trustProxy = old }(trustProxy)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.18.0.2:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	req.Header.Set("X-Forwarded-Proto", "https")
	trustProxy = false
	if clientIP(req) != "172.18.0.2" || isHTTPS(req) {
		t.Fatal("forwarded headers must be ignored when proxy is not trusted")
	}
	trustProxy = true
	if clientIP(req) != "203.0.113.7" || !isHTTPS(req) {
		t.Fatalf("clientIP = %s, https = %v", clientIP(req), isHTTPS(req))
	}
	req.Header.Set("X-Forwarded-For", "not-an-ip")
	if clientIP(req) != "172.18.0.2" {
		t.Fatal("garbage header must fall back to the socket address")
	}
}

func TestHealthzIsPublic(t *testing.T) {
	srv, _, _ := newAPI(t)
	r, err := http.Get(srv.URL + "/healthz")
	if err != nil || r.StatusCode != 200 {
		t.Fatalf("healthz = %v, %v", r, err)
	}
}

// mustGet fails the test on a transport error instead of letting a nil
// response panic later.
func mustGet(t *testing.T, cl *http.Client, url string) *http.Response {
	t.Helper()
	r, err := cl.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return r
}
