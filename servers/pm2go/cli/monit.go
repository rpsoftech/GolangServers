package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// monit is a full-screen terminal dashboard: process list, logs of the
// selected process and a host summary. Rendering is pure (render), key
// handling is pure (monitState.key), so tests need no TTY.

const (
	enterScreen = "\x1b[?1049h\x1b[?25l\x1b[2J"
	leaveScreen = "\x1b[0m\x1b[?25h\x1b[?1049l"
	logTailMax  = 200
	wideMinCols = 100 // below this the log pane goes under the list
	minCols     = 30
	minRows     = 6
)

// monitState is everything render needs.
type monitState struct {
	infos     []common.ProcessInfo
	sys       common.SystemSample
	haveSys   bool
	selID     int  // id of the selected process (-1 none)
	errLog    bool // show stderr instead of stdout
	filter    string
	filtering bool   // typing into the filter
	confirm   string // non-empty: waiting y/n to delete this id
	status    string // last error or message, shown on the bottom line
	statusErr bool
	logs      []string
	noColor   bool
}

// monitAction is what the loop must do after a key.
type monitAction struct {
	method string // daemon RPC, empty when none
	target string
	label  string // shown in the status line on success
	quit   bool
	redraw bool // reload logs and redraw (selection or toggle changed)
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07]*(\x07|\x1b\\)|\x1b.`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// clean makes a log line safe for the screen: no escapes, tabs or controls.
func clean(s string) string {
	s = stripANSI(s)
	s = strings.ReplaceAll(s, "\t", "    ")
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// fit truncates s to w runes and pads with spaces to exactly w.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	n := utf8.RuneCountInString(s)
	if n > w {
		r := []rune(s)
		if w > 1 {
			return string(r[:w-1]) + "…"
		}
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-n)
}

func (s *monitState) sty(code, text string) string {
	if s.noColor || code == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func statusColor(st string) string {
	switch st {
	case common.StatusOnline:
		return "32"
	case common.StatusErrored:
		return "31"
	case common.StatusStopped:
		return "90"
	}
	return "33"
}

func pctColor(p float64) string {
	switch {
	case p >= 85:
		return "31"
	case p >= 60:
		return "33"
	}
	return "32"
}

// bar draws pct (0..100) in w cells.
func (s *monitState) bar(pct float64, w int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	n := int(pct/100*float64(w) + 0.5)
	return s.sty(pctColor(pct), strings.Repeat("█", n)) + s.sty("90", strings.Repeat("░", w-n))
}

func memShort(b int64) string {
	switch {
	case b < 1<<10:
		return fmt.Sprintf("%dB", b)
	case b < 1<<20:
		return fmt.Sprintf("%dK", b>>10)
	case b < 1<<30:
		return fmt.Sprintf("%dM", b>>20)
	}
	return fmt.Sprintf("%.1fG", float64(b)/(1<<30))
}

// visible returns the processes matching the filter.
func (s *monitState) visible() []common.ProcessInfo {
	if s.filter == "" {
		return s.infos
	}
	f := strings.ToLower(s.filter)
	var out []common.ProcessInfo
	for _, i := range s.infos {
		if strings.Contains(strings.ToLower(i.Name), f) {
			out = append(out, i)
		}
	}
	return out
}

// selIndex is the index of the selection in the visible list (0 when unset).
func (s *monitState) selIndex() int {
	for n, i := range s.visible() {
		if i.ID == s.selID {
			return n
		}
	}
	return 0
}

func (s *monitState) selected() (common.ProcessInfo, bool) {
	v := s.visible()
	if len(v) == 0 {
		return common.ProcessInfo{}, false
	}
	return v[s.selIndex()], true
}

func (s *monitState) move(d int) {
	v := s.visible()
	if len(v) == 0 {
		return
	}
	n := s.selIndex() + d
	if n < 0 {
		n = 0
	}
	if n >= len(v) {
		n = len(v) - 1
	}
	s.selID = v[n].ID
}

func (s *monitState) setStatus(msg string, isErr bool) { s.status, s.statusErr = msg, isErr }

// key applies one key (see parseKeys) and says what to do next.
func (s *monitState) key(k string) monitAction {
	if k == "ctrl-c" {
		return monitAction{quit: true}
	}
	if s.confirm != "" {
		id := s.confirm
		s.confirm = ""
		if k == "y" || k == "Y" {
			return monitAction{method: common.MethodDelete, target: id, label: "deleted " + id}
		}
		s.setStatus("delete cancelled", false)
		return monitAction{redraw: true}
	}
	if s.filtering {
		switch k {
		case "enter":
			s.filtering = false
		case "esc":
			s.filtering, s.filter = false, ""
		case "backspace":
			if r := []rune(s.filter); len(r) > 0 {
				s.filter = string(r[:len(r)-1])
			}
		default:
			if utf8.RuneCountInString(k) == 1 && k[0] >= 0x20 {
				s.filter += k
			}
		}
		s.fixSelection()
		return monitAction{redraw: true}
	}
	s.setStatus("", false)
	p, have := s.selected()
	target := strconv.Itoa(p.ID)
	switch k {
	case "q":
		return monitAction{quit: true}
	case "up", "k":
		s.move(-1)
	case "down", "j":
		s.move(1)
	case "/":
		s.filtering = true
	case "esc":
		s.filter = ""
		s.fixSelection()
	case "e":
		s.errLog = !s.errLog
	case "r", "s", "f", "d":
		if !have {
			s.setStatus("no process selected", true)
			return monitAction{redraw: true}
		}
		switch k {
		case "r":
			return monitAction{method: common.MethodRestart, target: target, label: "restarted " + p.Name}
		case "s":
			return monitAction{method: common.MethodStop, target: target, label: "stopped " + p.Name}
		case "f":
			return monitAction{method: common.MethodFlush, target: target, label: "flushed logs of " + p.Name}
		}
		s.confirm = target
	default:
		return monitAction{}
	}
	return monitAction{redraw: true}
}

// fixSelection keeps the selection valid after the visible list changes.
func (s *monitState) fixSelection() {
	if v := s.visible(); len(v) > 0 {
		s.selID = v[s.selIndex()].ID
	}
}

// update stores a fresh process list.
func (s *monitState) update(infos []common.ProcessInfo) {
	s.infos = infos
	s.fixSelection()
}

// parseKeys splits one terminal read into key names.
func parseKeys(b []byte) []string {
	var out []string
	for len(b) > 0 {
		switch {
		case b[0] == 0x1b && len(b) == 1:
			out, b = append(out, "esc"), nil
		case b[0] == 0x1b && len(b) >= 3 && (b[1] == '[' || b[1] == 'O'):
			switch b[2] {
			case 'A':
				out = append(out, "up")
			case 'B':
				out = append(out, "down")
			}
			i := 2
			for i < len(b)-1 && (b[i] < '@' || b[i] > '~') { // skip the rest of the sequence
				i++
			}
			b = b[i+1:]
		case b[0] == 0x1b:
			b = b[1:]
		case b[0] == 3:
			out, b = append(out, "ctrl-c"), b[1:]
		case b[0] == '\r' || b[0] == '\n':
			out, b = append(out, "enter"), b[1:]
		case b[0] == 0x7f || b[0] == 8:
			out, b = append(out, "backspace"), b[1:]
		default:
			r, n := utf8.DecodeRune(b)
			if r >= 0x20 {
				out = append(out, string(r))
			}
			b = b[n:]
		}
	}
	return out
}

// hostLine is the top summary line.
func (s *monitState) hostLine() string {
	online := 0
	for _, i := range s.infos {
		if i.Status == common.StatusOnline {
			online++
		}
	}
	parts := []string{s.sty("1", "pm2go monit"), fmt.Sprintf("%d/%d online", online, len(s.infos))}
	if s.haveSys {
		h := s.sys
		parts = append(parts,
			fmt.Sprintf("cpu %.0f%%", h.CPU),
			fmt.Sprintf("mem %.0f%% (%s/%s)", h.MemPercent, memShort(h.MemUsed), memShort(h.MemTotal)),
			fmt.Sprintf("load %.2f %.2f %.2f", h.Load1, h.Load5, h.Load15),
			"up "+humanDuration(h.UptimeS*1000))
	}
	return strings.Join(parts, "  ")
}

// bottomLine is the status / help / prompt line.
func (s *monitState) bottomLine(w int) string {
	switch {
	case s.confirm != "":
		name := s.confirm
		for _, i := range s.infos {
			if strconv.Itoa(i.ID) == s.confirm {
				name = i.Name
			}
		}
		return s.sty("1;33", fit(fmt.Sprintf("delete %s? y/n", name), w))
	case s.filtering:
		return s.sty("1", fit("filter: "+s.filter+"█  (enter apply, esc clear)", w))
	case s.status != "":
		if s.statusErr {
			return s.sty("1;31", fit("error: "+s.status, w))
		}
		return s.sty("32", fit(s.status, w))
	}
	help := "↑↓ select  r restart  s stop  d delete  f flush  e out/err  / filter  q quit"
	if s.filter != "" {
		help = "filter \"" + s.filter + "\" (esc clears)  " + help
	}
	return s.sty("90", fit(help, w))
}

// listPane renders the process list into h lines of exactly w cells.
func (s *monitState) listPane(w, h int) []string {
	v := s.visible()
	title := fmt.Sprintf("PROCESSES %d", len(v))
	if s.filter != "" {
		title += fmt.Sprintf("/%d", len(s.infos))
	}
	lines := []string{s.sty("1;36", fit(title, w))}
	if h <= 1 {
		return lines[:h]
	}
	barW := 0
	if w >= 52 {
		barW = 8
	}
	// columns: id, name, status, cpu%, bar, mem
	idW, stW, cpuW, memW := 3, 8, 4, 6
	nameW := w - (idW + 1 + 1 + stW + 1 + cpuW + 1 + memW)
	if barW > 0 {
		nameW -= barW + 1
	}
	if nameW < 4 { // very narrow: drop status text to the badge initial
		nameW = 4
	}
	rows := h - 1
	first := 0
	if si := s.selIndex(); si >= rows {
		first = si - rows + 1
	}
	if len(v) == 0 {
		msg := "no processes"
		if s.filter != "" {
			msg = "no match"
		}
		lines = append(lines, s.sty("90", fit(msg, w)))
	}
	for n := first; n < len(v) && len(lines) < h; n++ {
		i := v[n]
		name := i.Name
		if i.Config.Instances > 1 || i.Instance > 0 {
			name += fmt.Sprintf("#%d", i.Instance)
		}
		cpu := fmt.Sprintf("%3.0f%%", i.CPU)
		cells := fit(strconv.Itoa(i.ID), idW) + " " + fit(name, nameW) + " " +
			s.sty(statusColor(i.Status), fit(i.Status, stW)) + " " + s.sty(pctColor(i.CPU), cpu)
		if barW > 0 {
			cells += " " + s.bar(i.CPU, barW)
		}
		cells += " " + fmt.Sprintf("%*s", memW, memShort(i.MemoryB))
		if n == s.selIndex() {
			cells = s.sty("7", stripANSI(cells))
			if s.noColor {
				cells = ">" + stripANSI(cells)[1:]
			}
		}
		lines = append(lines, fitANSI(cells, w))
	}
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return lines
}

// fitANSI makes sure a line built from fitted cells is exactly w wide.
func fitANSI(s string, w int) string {
	plain := stripANSI(s)
	n := utf8.RuneCountInString(plain)
	switch {
	case n == w:
		return s
	case n < w:
		return s + strings.Repeat(" ", w-n)
	}
	return fit(plain, w)
}

// logPane renders the tail of the selected log into h lines of w cells.
func (s *monitState) logPane(w, h int) []string {
	title := "LOGS"
	if p, ok := s.selected(); ok {
		kind := "out"
		if s.errLog {
			kind = "err"
		}
		title = fmt.Sprintf("LOGS %s [%s]", p.Name, kind)
	}
	lines := []string{s.sty("1;36", fit(title, w))}
	body := s.logs
	if rows := h - 1; len(body) > rows {
		if rows < 0 {
			rows = 0
		}
		body = body[len(body)-rows:]
	}
	for _, l := range body {
		if len(lines) >= h {
			break
		}
		lines = append(lines, fit(clean(l), w))
	}
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return lines[:h]
}

// render draws the whole screen as w*h cells joined by "\n" (no trailing newline).
func render(s *monitState, w, h int) string {
	if w < minCols || h < minRows {
		return fit(fmt.Sprintf("terminal too small (%dx%d, need %dx%d)", w, h, minCols, minRows), w)
	}
	out := []string{fit(stripANSI(s.hostLine()), w)}
	if !s.noColor {
		out[0] = s.sty("7", out[0])
	}
	body := h - 2
	if w >= wideMinCols {
		lw := w * 45 / 100
		if lw > 70 {
			lw = 70
		}
		l, r := s.listPane(lw, body), s.logPane(w-lw-1, body)
		for n := 0; n < body; n++ {
			out = append(out, l[n]+s.sty("90", "│")+r[n])
		}
	} else {
		lh := len(s.visible()) + 1
		if lh > body/2 {
			lh = body / 2
		}
		if lh < 3 {
			lh = 3
		}
		out = append(out, s.listPane(w, lh)...)
		out = append(out, s.logPane(w, body-lh)...)
	}
	out = append(out, s.bottomLine(w))
	return strings.Join(out, "\n")
}

// frame turns a rendered screen into one terminal write (no flicker: home,
// overwrite every row, clear the rest).
func frame(screen string) string {
	return "\x1b[H" + strings.ReplaceAll(screen, "\n", "\x1b[K\r\n") + "\x1b[K\x1b[J"
}

// monitIO is what the loop needs from the outside, so tests can fake it.
type monitIO struct {
	keys  <-chan []byte
	out   io.Writer
	size  func() (int, int)
	winch <-chan os.Signal
	quit  <-chan os.Signal
	tick  <-chan time.Time
}

// monitRPC makes one RPC call with a deadline and no daemon auto-spawn.
func monitRPC(method string, params any, out any) error {
	conn, err := connect(false)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	return roundTrip(conn, method, params, out)
}

func (s *monitState) refresh() {
	var infos []common.ProcessInfo
	if err := monitRPC(common.MethodList, nil, &infos); err != nil {
		s.setStatus("daemon: "+err.Error(), true)
		return
	}
	s.update(infos)
	var sys common.SystemSample
	s.haveSys = monitRPC(common.MethodSystem, nil, &sys) == nil
	if s.haveSys {
		s.sys = sys
	}
}

func (s *monitState) loadLogs() {
	s.logs = nil
	if p, ok := s.selected(); ok {
		path := p.OutLog
		if s.errLog {
			path = p.ErrLog
		}
		s.logs, _ = lastLines(path, logTailMax)
	}
}

type actionDone struct {
	label string
	err   error
}

// runMonit is the event loop. It returns when the user quits or on a quit signal.
func runMonit(s *monitState, env monitIO) {
	done := make(chan actionDone, 4)
	draw := func() {
		w, h := env.size()
		fmt.Fprint(env.out, frame(render(s, w, h)))
	}
	reload := func() {
		s.refresh()
		s.loadLogs()
		draw()
	}
	reload()
	for {
		select {
		case <-env.quit:
			return
		case <-env.winch:
			draw()
		case <-env.tick:
			reload()
		case d := <-done:
			if d.err != nil {
				s.setStatus(d.err.Error(), true)
			} else {
				s.setStatus(d.label, false)
			}
			reload()
		case b, ok := <-env.keys:
			if !ok {
				return
			}
			for _, k := range parseKeys(b) {
				a := s.key(k)
				if a.quit {
					return
				}
				if a.method != "" {
					go func() {
						var infos []common.ProcessInfo
						err := monitRPC(a.method, common.TargetParams{Target: a.target}, &infos)
						done <- actionDone{a.label, err}
					}()
				}
			}
			s.loadLogs()
			draw()
		}
	}
}

var errNoTTY = errors.New("monit needs an interactive terminal (stdin and stdout must be a TTY)")

// cmdMonit runs the dashboard on the real terminal.
func cmdMonit(args []string) error {
	if len(args) > 0 {
		return errors.New("monit takes no arguments")
	}
	in, out := int(os.Stdin.Fd()), int(os.Stdout.Fd())
	if !term.IsTerminal(in) || !term.IsTerminal(out) {
		return errNoTTY
	}
	if c, err := connect(false); err != nil {
		return errors.New("daemon is not running (start a process with pm2go start)")
	} else {
		c.Close()
	}
	old, err := term.MakeRaw(in)
	if err != nil {
		return err
	}
	var once sync.Once
	restore := func() {
		once.Do(func() {
			io.WriteString(os.Stdout, leaveScreen)
			term.Restore(in, old)
		})
	}
	defer restore()
	defer func() {
		if r := recover(); r != nil {
			restore()
			panic(r)
		}
	}()
	io.WriteString(os.Stdout, enterScreen)

	keys := make(chan []byte)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				close(keys)
				return
			}
			keys <- append([]byte(nil), buf[:n]...)
		}
	}()
	winch := make(chan os.Signal, 1)
	notifyResize(winch)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(winch)
	defer signal.Stop(quit)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	st := &monitState{selID: -1, noColor: os.Getenv("NO_COLOR") != ""}
	runMonit(st, monitIO{
		keys: keys, out: os.Stdout, winch: winch, quit: quit, tick: tick.C,
		size: func() (int, int) {
			w, h, err := term.GetSize(out)
			if err != nil {
				return 80, 24
			}
			return w, h
		},
	})
	return nil
}
