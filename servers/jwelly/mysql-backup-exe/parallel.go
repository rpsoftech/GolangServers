package main

import (
	"context"
	"sync"
)

// maxParallel caps the parallel setting; more connections than this only
// loads the database servers.
const maxParallel = 16

// runPool runs run for every database with at most n at the same time (n < 1
// means one at a time) and returns the results in the order of dbs. A
// database that was never started because ctx ended has a nil result.
func runPool(ctx context.Context, n int, dbs []*DatabaseConfig, run func(*DatabaseConfig) *Result) []*Result {
	if n < 1 {
		n = 1
	}
	results := make([]*Result, len(dbs))
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(n, len(dbs)) {
		wg.Go(func() {
			for i := range next {
				results[i] = run(dbs[i])
			}
		})
	}
	for i := range dbs {
		if ctx.Err() != nil {
			break
		}
		next <- i
	}
	close(next)
	wg.Wait()
	return results
}

// gate limits how many backups run at once in the daemon. The zero value
// (nil) lets everything through.
type gate chan struct{}

// newGate returns a gate for n at a time, or nil, meaning no limit, when n < 1.
func newGate(n int) gate {
	if n < 1 {
		return nil
	}
	return make(gate, n)
}

// acquire waits for a free slot. It returns false if ctx ended first.
func (g gate) acquire(ctx context.Context) bool {
	if g == nil {
		return true
	}
	select {
	case g <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (g gate) release() {
	if g != nil {
		<-g
	}
}
