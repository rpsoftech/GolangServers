//go:build linux

package daemon

import (
	"bufio"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const clkTck = 100.0 // USER_HZ is 100 on every Linux the project targets

type usage struct {
	cpu float64
	rss int64
}

// Sampler computes per-process-group CPU and memory from /proc. State is kept
// between calls to turn tick counters into percentages.
type Sampler struct {
	prevTicks map[int]float64 // pgrp -> total ticks
	prevAt    time.Time
	prevCPU   cpuTimes
	prevNet   netCounters
	prevNetAt time.Time
}

func NewSampler() *Sampler { return &Sampler{prevTicks: map[int]float64{}} }

type procStat struct {
	pgrp  int
	ticks float64
	rss   int64
}

func readProcStat(pid string) (procStat, bool) {
	b, err := os.ReadFile("/proc/" + pid + "/stat")
	if err != nil {
		return procStat{}, false
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 || i+2 >= len(s) {
		return procStat{}, false
	}
	f := strings.Fields(s[i+2:]) // starts at field 3 (state)
	if len(f) < 15 {
		return procStat{}, false
	}
	pgrp, _ := strconv.Atoi(f[2])
	ut, _ := strconv.ParseFloat(f[11], 64)
	st, _ := strconv.ParseFloat(f[12], 64)
	var rss int64
	if sm, err := os.ReadFile("/proc/" + pid + "/statm"); err == nil {
		ff := strings.Fields(string(sm))
		if len(ff) >= 2 {
			pages, _ := strconv.ParseInt(ff[1], 10, 64)
			rss = pages * int64(os.Getpagesize())
		}
	}
	return procStat{pgrp: pgrp, ticks: ut + st, rss: rss}, true
}

// SampleProcs returns usage per tracked leader pid (sum over its process group).
func (s *Sampler) SampleProcs(pids []int) map[int]usage {
	out := map[int]usage{}
	if len(pids) == 0 {
		s.prevTicks = map[int]float64{}
		return out
	}
	want := map[int]bool{}
	for _, p := range pids {
		want[p] = true
	}
	ticks := map[int]float64{}
	rss := map[int]int64{}
	ents, _ := filepath.Glob("/proc/[0-9]*")
	for _, e := range ents {
		ps, ok := readProcStat(filepath.Base(e))
		if !ok || !want[ps.pgrp] {
			continue
		}
		ticks[ps.pgrp] += ps.ticks
		rss[ps.pgrp] += ps.rss
	}
	now := time.Now()
	dt := now.Sub(s.prevAt).Seconds()
	for _, p := range pids {
		u := usage{rss: rss[p]}
		if prev, ok := s.prevTicks[p]; ok && dt > 0 && ticks[p] >= prev {
			u.cpu = (ticks[p] - prev) / clkTck / dt * 100
		}
		out[p] = u
	}
	s.prevTicks, s.prevAt = ticks, now
	return out
}

type cpuTimes struct{ idle, total float64 }
type netCounters struct{ rx, tx uint64 }

type SystemSample = common.SystemSample

func readCPUTimes() (cpuTimes, bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTimes{}, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return cpuTimes{}, false
	}
	fs := strings.Fields(sc.Text())
	if len(fs) < 5 || fs[0] != "cpu" {
		return cpuTimes{}, false
	}
	var total, idle float64
	for i, v := range fs[1:] {
		n, _ := strconv.ParseFloat(v, 64)
		if i < 8 { // user..steal; guest fields are already inside user
			total += n
		}
		if i == 3 || i == 4 { // idle + iowait
			idle += n
		}
	}
	return cpuTimes{idle: idle, total: total}, true
}

func readNet() netCounters {
	var c netCounters
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return c
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		i := strings.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		if strings.TrimSpace(line[:i]) == "lo" {
			continue
		}
		fs := strings.Fields(line[i+1:])
		if len(fs) < 9 {
			continue
		}
		rx, _ := strconv.ParseUint(fs[0], 10, 64)
		tx, _ := strconv.ParseUint(fs[8], 10, 64)
		c.rx += rx
		c.tx += tx
	}
	return c
}

func (s *Sampler) SampleSystem() SystemSample {
	var out SystemSample
	out.CPUCount = cpuCount()
	if t, ok := readCPUTimes(); ok {
		if s.prevCPU.total > 0 && t.total > s.prevCPU.total {
			out.CPU = (1 - (t.idle-s.prevCPU.idle)/(t.total-s.prevCPU.total)) * 100
		}
		s.prevCPU = t
	}
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, avail int64
		for _, l := range strings.Split(string(b), "\n") {
			fs := strings.Fields(l)
			if len(fs) < 2 {
				continue
			}
			kb, _ := strconv.ParseInt(fs[1], 10, 64)
			switch fs[0] {
			case "MemTotal:":
				total = kb * 1024
			case "MemAvailable:":
				avail = kb * 1024
			}
		}
		out.MemTotal, out.MemUsed = total, total-avail
		if total > 0 {
			out.MemPercent = float64(out.MemUsed) / float64(total) * 100
		}
	}
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		fs := strings.Fields(string(b))
		if len(fs) >= 3 {
			out.Load1, _ = strconv.ParseFloat(fs[0], 64)
			out.Load5, _ = strconv.ParseFloat(fs[1], 64)
			out.Load15, _ = strconv.ParseFloat(fs[2], 64)
		}
	}
	if b, err := os.ReadFile("/proc/uptime"); err == nil {
		fs := strings.Fields(string(b))
		if len(fs) > 0 {
			up, _ := strconv.ParseFloat(fs[0], 64)
			out.UptimeS = int64(up)
		}
	}
	var st syscall.Statfs_t
	if syscall.Statfs("/", &st) == nil {
		out.DiskTotal = int64(st.Blocks) * int64(st.Bsize)
		out.DiskUsed = int64(st.Blocks-st.Bfree) * int64(st.Bsize)
		if out.DiskTotal > 0 {
			out.DiskPct = float64(out.DiskUsed) / float64(out.DiskTotal) * 100
		}
	}
	n := readNet()
	now := time.Now()
	if !s.prevNetAt.IsZero() {
		dt := now.Sub(s.prevNetAt).Seconds()
		if dt > 0 && n.rx >= s.prevNet.rx && n.tx >= s.prevNet.tx {
			out.NetRxBps = float64(n.rx-s.prevNet.rx) / dt
			out.NetTxBps = float64(n.tx-s.prevNet.tx) / dt
		}
	}
	s.prevNet, s.prevNetAt = n, now
	return out
}

func cpuCount() int { return runtime.NumCPU() }
