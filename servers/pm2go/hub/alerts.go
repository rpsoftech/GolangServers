package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// Rule types.
const (
	TypeSystemCPU    = "system_cpu"    // percent
	TypeSystemMemory = "system_memory" // percent
	TypeSystemDisk   = "system_disk"   // percent
	TypeSystemLoad   = "system_load"   // load1 per CPU core x100 (100 = one runnable task per core)
	TypeProcCPU      = "proc_cpu"      // percent of one core, summed over instances
	TypeProcMemory   = "proc_memory"   // MB, summed over instances
	TypeProcStatus   = "proc_status"   // an instance is errored (crash loop gave up)
	TypeRestartLoop  = "restart_loop"  // Count crashes within WindowS
	TypeLogErrors    = "log_errors"    // Count matching log lines within WindowS
	TypeServerOffl   = "server_offline"
)

// Rule is one alert definition. Several rules of the same type with different
// levels give escalation: the highest firing level wins the notification.
type Rule struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	Level          string   `json:"level"`
	Type           string   `json:"type"`
	Server         string   `json:"server"`  // "" or "*" = all
	Process        string   `json:"process"` // "" or "*" = all
	Threshold      float64  `json:"threshold"`
	DurationS      int      `json:"duration_s"` // condition must hold this long before firing
	WindowS        int      `json:"window_s"`
	Count          int      `json:"count"`
	Pattern        string   `json:"pattern"`
	Stream         string   `json:"stream"`   // "", out, err
	RepeatS        int      `json:"repeat_s"` // remind while still firing; 0 = never
	NotifyResolved bool     `json:"notify_resolved"`
	Chats          []string `json:"chats"` // chat ids; empty = all eligible chats
	re             *regexp.Regexp
}

func (r *Rule) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if !validLevel(r.Level) {
		return fmt.Errorf("level must be info, warning or critical")
	}
	switch r.Type {
	case TypeSystemCPU, TypeSystemMemory, TypeSystemDisk, TypeSystemLoad, TypeProcCPU, TypeProcMemory:
		if r.Threshold <= 0 {
			return errors.New("threshold must be > 0")
		}
	case TypeRestartLoop, TypeLogErrors:
		if r.Count <= 0 || r.WindowS <= 0 {
			return errors.New("count and window_s must be > 0")
		}
		if r.Type == TypeLogErrors && r.Pattern != "" {
			if _, err := regexp.Compile(r.Pattern); err != nil {
				return fmt.Errorf("pattern: %w", err)
			}
		}
	case TypeProcStatus, TypeServerOffl:
	default:
		return fmt.Errorf("unknown type %q", r.Type)
	}
	if r.Stream != "" && r.Stream != "out" && r.Stream != "err" {
		return errors.New("stream must be out, err or empty")
	}
	if r.DurationS < 0 || r.RepeatS < 0 {
		return errors.New("duration_s and repeat_s must be >= 0")
	}
	return nil
}

func matches(pattern, s string) bool { return pattern == "" || pattern == "*" || pattern == s }

// AlertState is one tracked (rule, server, process) condition.
type AlertState struct {
	RuleID    int64   `json:"rule_id"`
	RuleName  string  `json:"rule_name"`
	Level     string  `json:"level"`
	Type      string  `json:"type"`
	Server    string  `json:"server"`
	Process   string  `json:"process"`
	State     string  `json:"state"` // pending, firing
	Since     int64   `json:"since"` // unix ms the condition became true
	FiredAt   int64   `json:"fired_at,omitempty"`
	Value     float64 `json:"value"`
	Detail    string  `json:"detail"`
	Suppress  bool    `json:"suppressed"` // a higher level alert covers this one
	lastNote  time.Time
	notified  bool
	falseFrom time.Time
	rule      *Rule
}

const recoverAfter = 10 * time.Second

// Engine evaluates rules against registry state on a timer.
type Engine struct {
	store *Store
	reg   *Registry
	notif *Notifier
	now   func() time.Time

	mu     sync.Mutex
	rules  []*Rule
	states map[string]*AlertState
}

func NewEngine(store *Store, reg *Registry, n *Notifier) (*Engine, error) {
	e := &Engine{store: store, reg: reg, notif: n, now: time.Now, states: map[string]*AlertState{}}
	if err := e.loadRules(); err != nil {
		return nil, err
	}
	if len(e.rules) == 0 {
		if err := e.seedDefaults(); err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e *Engine) loadRules() error {
	rows, err := e.store.db.Query(`SELECT id, body FROM alert_rules ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var rules []*Rule
	for rows.Next() {
		var id int64
		var body string
		if err := rows.Scan(&id, &body); err != nil {
			return err
		}
		var r Rule
		if json.Unmarshal([]byte(body), &r) != nil {
			continue
		}
		r.ID = id
		compileRule(&r)
		rules = append(rules, &r)
	}
	e.mu.Lock()
	e.rules = rules
	e.mu.Unlock()
	return rows.Err()
}

func compileRule(r *Rule) {
	p := r.Pattern
	if p == "" {
		p = errorLine.String()
	}
	r.re, _ = regexp.Compile(p)
}

func (e *Engine) seedDefaults() error {
	defs := []Rule{
		{Name: "Process crashed and gave up", Level: "critical", Type: TypeProcStatus, NotifyResolved: true},
		{Name: "Restart loop", Level: "warning", Type: TypeRestartLoop, Count: 3, WindowS: 300, NotifyResolved: true},
		{Name: "Restart loop (severe)", Level: "critical", Type: TypeRestartLoop, Count: 10, WindowS: 300, NotifyResolved: true},
		{Name: "Process CPU high", Level: "warning", Type: TypeProcCPU, Threshold: 80, DurationS: 300, RepeatS: 1800, NotifyResolved: true},
		{Name: "Process CPU very high", Level: "critical", Type: TypeProcCPU, Threshold: 95, DurationS: 120, RepeatS: 900, NotifyResolved: true},
		{Name: "Server memory high", Level: "warning", Type: TypeSystemMemory, Threshold: 85, DurationS: 300, RepeatS: 1800, NotifyResolved: true},
		{Name: "Server memory critical", Level: "critical", Type: TypeSystemMemory, Threshold: 95, DurationS: 120, RepeatS: 900, NotifyResolved: true},
		{Name: "Server CPU high", Level: "warning", Type: TypeSystemCPU, Threshold: 85, DurationS: 600, RepeatS: 1800, NotifyResolved: true},
		{Name: "Disk filling up", Level: "warning", Type: TypeSystemDisk, Threshold: 85, DurationS: 60, RepeatS: 3600, NotifyResolved: true},
		{Name: "Disk almost full", Level: "critical", Type: TypeSystemDisk, Threshold: 95, DurationS: 60, RepeatS: 900, NotifyResolved: true},
		{Name: "Server offline", Level: "critical", Type: TypeServerOffl, DurationS: 60, RepeatS: 900, NotifyResolved: true},
		{Name: "Error burst", Level: "warning", Type: TypeLogErrors, Count: 10, WindowS: 60, NotifyResolved: true},
		{Name: "Error flood", Level: "critical", Type: TypeLogErrors, Count: 100, WindowS: 60, NotifyResolved: true},
	}
	for _, d := range defs {
		d.Enabled = true
		if _, err := e.AddRule(d); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) Rules() []Rule {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Rule, len(e.rules))
	for i, r := range e.rules {
		out[i] = *r
	}
	return out
}

func (e *Engine) AddRule(r Rule) (Rule, error) {
	if err := r.Validate(); err != nil {
		return r, err
	}
	b, _ := json.Marshal(r)
	res, err := e.store.db.Exec(`INSERT INTO alert_rules(body) VALUES(?)`, string(b))
	if err != nil {
		return r, err
	}
	r.ID, _ = res.LastInsertId()
	compileRule(&r)
	e.mu.Lock()
	e.rules = append(e.rules, &r)
	e.mu.Unlock()
	return r, nil
}

func (e *Engine) UpdateRule(id int64, r Rule) (Rule, error) {
	r.ID = id
	if err := r.Validate(); err != nil {
		return r, err
	}
	b, _ := json.Marshal(r)
	res, err := e.store.db.Exec(`UPDATE alert_rules SET body=? WHERE id=?`, string(b), id)
	if err != nil {
		return r, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return r, errors.New("rule not found")
	}
	compileRule(&r)
	e.mu.Lock()
	for i, old := range e.rules {
		if old.ID == id {
			e.rules[i] = &r
		}
	}
	e.dropStates(id)
	e.mu.Unlock()
	return r, nil
}

func (e *Engine) DeleteRule(id int64) error {
	if _, err := e.store.db.Exec(`DELETE FROM alert_rules WHERE id=?`, id); err != nil {
		return err
	}
	e.mu.Lock()
	kept := e.rules[:0]
	for _, r := range e.rules {
		if r.ID != id {
			kept = append(kept, r)
		}
	}
	e.rules = kept
	e.dropStates(id)
	e.mu.Unlock()
	return nil
}

// dropStates forgets live state of a rule. Caller holds e.mu.
func (e *Engine) dropStates(id int64) {
	for k, s := range e.states {
		if s.RuleID == id {
			delete(e.states, k)
		}
	}
}

// Active returns pending and firing alerts, most severe first.
func (e *Engine) Active() []AlertState {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]AlertState, 0, len(e.states))
	for _, s := range e.states {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if levelRank[out[i].Level] != levelRank[out[j].Level] {
			return levelRank[out[i].Level] > levelRank[out[j].Level]
		}
		return out[i].Since > out[j].Since
	})
	return out
}

type observation struct {
	server, proc string
	hit          bool
	value        float64
	detail       string
}

// Run evaluates on a ticker until stop closes.
func (e *Engine) Run(every time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			e.Evaluate()
		}
	}
}

// Evaluate runs one pass over all enabled rules.
func (e *Engine) Evaluate() {
	e.mu.Lock()
	rules := make([]*Rule, 0, len(e.rules))
	for _, r := range e.rules {
		if r.Enabled {
			rules = append(rules, r)
		}
	}
	e.mu.Unlock()
	servers := e.reg.States()
	now := e.now()
	seen := map[string]bool{}
	for _, r := range rules {
		for _, o := range e.observe(r, servers, now) {
			key := fmt.Sprintf("%d|%s|%s", r.ID, o.server, o.proc)
			seen[key] = true
			e.apply(r, key, o, now)
		}
	}
	// states whose rule vanished or is disabled, or whose target disappeared
	e.mu.Lock()
	var gone []*AlertState
	for k, s := range e.states {
		if !seen[k] {
			gone = append(gone, s)
		}
	}
	e.mu.Unlock()
	for _, s := range gone {
		key := fmt.Sprintf("%d|%s|%s", s.RuleID, s.Server, s.Process)
		e.apply(s.rule, key, observation{server: s.Server, proc: s.Process, hit: false}, now)
	}
	e.markSuppression()
}

func (e *Engine) observe(r *Rule, servers []ServerState, now time.Time) []observation {
	var out []observation
	for _, sv := range servers {
		if !matches(r.Server, sv.Name) {
			continue
		}
		switch r.Type {
		case TypeServerOffl:
			ago := now.Sub(time.UnixMilli(sv.LastSeen))
			hit := !sv.Online && ago > 0
			out = append(out, observation{server: sv.Name, hit: hit, value: ago.Seconds(),
				detail: fmt.Sprintf("no data for %s", ago.Round(time.Second))})
			continue
		}
		if !sv.Online || sv.Snapshot == nil {
			continue // offline hosts are covered by the offline rule
		}
		sy := sv.Snapshot.System
		switch r.Type {
		case TypeSystemCPU:
			out = append(out, observation{server: sv.Name, hit: sy.CPU > r.Threshold, value: sy.CPU,
				detail: fmt.Sprintf("CPU %.1f%% (threshold %.0f%%)", sy.CPU, r.Threshold)})
		case TypeSystemMemory:
			out = append(out, observation{server: sv.Name, hit: sy.MemPercent > r.Threshold, value: sy.MemPercent,
				detail: fmt.Sprintf("memory %.1f%% of %s (threshold %.0f%%)", sy.MemPercent, humanBytes(sy.MemTotal), r.Threshold)})
		case TypeSystemDisk:
			out = append(out, observation{server: sv.Name, hit: sy.DiskPct > r.Threshold, value: sy.DiskPct,
				detail: fmt.Sprintf("disk %.1f%% of %s (threshold %.0f%%)", sy.DiskPct, humanBytes(sy.DiskTotal), r.Threshold)})
		case TypeSystemLoad:
			per := sy.Load1 / float64(max(1, sy.CPUCount)) * 100
			out = append(out, observation{server: sv.Name, hit: per > r.Threshold, value: per,
				detail: fmt.Sprintf("load1 %.2f on %d cores (%.0f%% per core, threshold %.0f%%)", sy.Load1, sy.CPUCount, per, r.Threshold)})
		case TypeProcCPU, TypeProcMemory, TypeProcStatus, TypeRestartLoop, TypeLogErrors:
			out = append(out, e.observeProcs(r, sv, now)...)
		}
	}
	return out
}

type agg struct {
	cpu     float64
	mem     int64
	errored int
	count   int
}

func (e *Engine) observeProcs(r *Rule, sv ServerState, now time.Time) []observation {
	byName := map[string]*agg{}
	for _, p := range sv.Snapshot.Processes {
		if !matches(r.Process, p.Name) {
			continue
		}
		a := byName[p.Name]
		if a == nil {
			a = &agg{}
			byName[p.Name] = a
		}
		a.cpu += p.CPU
		a.mem += p.MemoryB
		a.count++
		if p.Status == common.StatusErrored {
			a.errored++
		}
	}
	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []observation
	for _, n := range names {
		a := byName[n]
		o := observation{server: sv.Name, proc: n}
		switch r.Type {
		case TypeProcCPU:
			o.hit, o.value = a.cpu > r.Threshold, a.cpu
			o.detail = fmt.Sprintf("CPU %.1f%% across %d instance(s) (threshold %.0f%%)", a.cpu, a.count, r.Threshold)
		case TypeProcMemory:
			mb := float64(a.mem) / (1 << 20)
			o.hit, o.value = mb > r.Threshold, mb
			o.detail = fmt.Sprintf("memory %.0f MB across %d instance(s) (threshold %.0f MB)", mb, a.count, r.Threshold)
		case TypeProcStatus:
			o.hit, o.value = a.errored > 0, float64(a.errored)
			o.detail = fmt.Sprintf("%d of %d instance(s) errored", a.errored, a.count)
		case TypeRestartLoop:
			c := e.reg.CrashCount(sv.Name, n, time.Duration(r.WindowS)*time.Second)
			o.hit, o.value = c >= r.Count, float64(c)
			o.detail = fmt.Sprintf("%d crashes in %s (threshold %d)", c, time.Duration(r.WindowS)*time.Second, r.Count)
		case TypeLogErrors:
			c := 0
			for _, l := range e.reg.LogsSince(sv.Name, n, now.Add(-time.Duration(r.WindowS)*time.Second)) {
				if r.Stream != "" && l.Stream != r.Stream {
					continue
				}
				if r.re != nil && r.re.MatchString(l.Line) {
					c++
				}
			}
			o.hit, o.value = c >= r.Count, float64(c)
			o.detail = fmt.Sprintf("%d matching log lines in %s (threshold %d)", c, time.Duration(r.WindowS)*time.Second, r.Count)
		}
		out = append(out, o)
	}
	return out
}

// apply advances the state machine for one (rule, target).
func (e *Engine) apply(r *Rule, key string, o observation, now time.Time) {
	e.mu.Lock()
	s := e.states[key]
	if o.hit {
		if s == nil {
			s = &AlertState{RuleID: r.ID, RuleName: r.Name, Level: r.Level, Type: r.Type,
				Server: o.server, Process: o.proc, State: "pending", Since: now.UnixMilli(), rule: r}
			e.states[key] = s
		}
		s.falseFrom = time.Time{}
		s.Value, s.Detail, s.rule, s.RuleName, s.Level = o.value, o.detail, r, r.Name, r.Level
		if s.State == "pending" && now.Sub(time.UnixMilli(s.Since)) >= time.Duration(r.DurationS)*time.Second {
			s.State, s.FiredAt = "firing", now.UnixMilli()
		}
		e.mu.Unlock()
		return
	}
	if s == nil {
		e.mu.Unlock()
		return
	}
	if s.State == "pending" {
		delete(e.states, key)
		e.mu.Unlock()
		return
	}
	if s.falseFrom.IsZero() {
		s.falseFrom = now
	}
	if now.Sub(s.falseFrom) < recoverAfter {
		e.mu.Unlock()
		return
	}
	delete(e.states, key)
	rule := s.rule
	notified := s.notified
	st := *s
	e.mu.Unlock()
	notify := rule != nil && rule.NotifyResolved && notified
	if notify {
		e.notif.Notify(st.Level, FormatAlert("resolved", st.Level, st.RuleName, st.Server, st.Process,
			fmt.Sprintf("was firing for %s", now.Sub(time.UnixMilli(st.FiredAt)).Round(time.Second))), rule.Chats)
	}
	e.record(st, "resolved", notify)
}

// markSuppression flags lower-level firing alerts covered by a higher one for
// the same target and type, then sends due notifications and reminders.
func (e *Engine) markSuppression() {
	now := e.now()
	type send struct {
		s    AlertState
		rule *Rule
		rem  bool
	}
	var sends []send
	e.mu.Lock()
	top := map[string]int{}
	for _, s := range e.states {
		if s.State != "firing" {
			continue
		}
		g := s.Type + "|" + s.Server + "|" + s.Process
		if levelRank[s.Level] > top[g] {
			top[g] = levelRank[s.Level]
		}
	}
	for _, s := range e.states {
		if s.State != "firing" {
			s.Suppress = false
			continue
		}
		g := s.Type + "|" + s.Server + "|" + s.Process
		s.Suppress = levelRank[s.Level] < top[g]
		if s.Suppress || s.rule == nil {
			continue
		}
		if !s.notified {
			s.notified, s.lastNote = true, now
			sends = append(sends, send{s: *s, rule: s.rule})
		} else if s.rule.RepeatS > 0 && now.Sub(s.lastNote) >= time.Duration(s.rule.RepeatS)*time.Second {
			s.lastNote = now
			sends = append(sends, send{s: *s, rule: s.rule, rem: true})
		}
	}
	e.mu.Unlock()
	for _, x := range sends {
		detail := x.s.Detail
		if x.rem {
			detail = fmt.Sprintf("still firing after %s: %s", now.Sub(time.UnixMilli(x.s.FiredAt)).Round(time.Second), detail)
		}
		n := e.notif.Notify(x.s.Level, FormatAlert("firing", x.s.Level, x.s.RuleName, x.s.Server, x.s.Process, detail), x.rule.Chats)
		if !x.rem {
			e.record(x.s, "firing", n > 0)
		}
	}
}

func (e *Engine) record(s AlertState, state string, notified bool) {
	n := 0
	if notified {
		n = 1
	}
	_, err := e.store.db.Exec(`INSERT INTO alert_history(ts,rule_id,rule_name,level,server,proc,state,message,value,notified) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		e.now().UnixMilli(), s.RuleID, s.RuleName, s.Level, s.Server, s.Process, state, s.Detail, s.Value, n)
	if err != nil {
		log.Printf("record alert: %v", err)
	}
}

func humanBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(b)/(1<<20))
	}
	return fmt.Sprintf("%d B", b)
}

// History returns recent alert transitions, newest first.
func (e *Engine) History(limit int) ([]map[string]any, error) {
	rows, err := e.store.db.Query(`SELECT id,ts,rule_id,rule_name,level,server,proc,state,message,value,notified FROM alert_history ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, ts, ruleID, notified int64
		var name, level, server, proc, state, msg string
		var val float64
		if err := rows.Scan(&id, &ts, &ruleID, &name, &level, &server, &proc, &state, &msg, &val, &notified); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "ts": ts, "rule_id": ruleID, "rule_name": name, "level": level,
			"server": server, "process": proc, "state": state, "message": msg, "value": val, "notified": notified == 1})
	}
	return out, rows.Err()
}
