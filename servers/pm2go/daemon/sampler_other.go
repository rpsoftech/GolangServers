//go:build !linux && !windows

package daemon

import (
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Non-Linux builds exist only so the code can be developed and tested on a
// laptop. Production targets Linux and uses /proc (sampler_linux.go).

type usage struct {
	cpu float64
	rss int64
}

type Sampler struct{}

func NewSampler() *Sampler { return &Sampler{} }

type SystemSample = common.SystemSample

func (s *Sampler) SampleSystem() SystemSample {
	out := SystemSample{CPUCount: 1}
	var st syscall.Statfs_t
	if syscall.Statfs("/", &st) == nil {
		out.DiskTotal = int64(st.Blocks) * int64(st.Bsize)
		out.DiskUsed = int64(st.Blocks-st.Bfree) * int64(st.Bsize)
		if out.DiskTotal > 0 {
			out.DiskPct = float64(out.DiskUsed) / float64(out.DiskTotal) * 100
		}
	}
	return out
}

func (s *Sampler) SampleProcs(pids []int) map[int]usage {
	out := map[int]usage{}
	if len(pids) == 0 {
		return out
	}
	list := make([]string, len(pids))
	for i, p := range pids {
		list[i] = strconv.Itoa(p)
	}
	b, err := exec.Command("ps", "-o", "pid=,rss=,pcpu=", "-p", strings.Join(list, ",")).Output()
	if err != nil && len(b) == 0 {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		pid, e1 := strconv.Atoi(f[0])
		rss, e2 := strconv.ParseInt(f[1], 10, 64)
		cpu, e3 := strconv.ParseFloat(f[2], 64)
		if e1 == nil && e2 == nil && e3 == nil {
			out[pid] = usage{cpu: cpu, rss: rss * 1024}
		}
	}
	return out
}
