//go:build windows

package daemon

import (
	"runtime"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// Development stub: Windows reports no usage numbers.

type usage struct {
	cpu float64
	rss int64
}

type Sampler struct{}

func NewSampler() *Sampler { return &Sampler{} }

type SystemSample = common.SystemSample

func (s *Sampler) SampleSystem() SystemSample { return SystemSample{CPUCount: runtime.NumCPU()} }

func (s *Sampler) SampleProcs(pids []int) map[int]usage { return map[int]usage{} }
