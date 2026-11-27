package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func namedDBs(n int) []*DatabaseConfig {
	dbs := make([]*DatabaseConfig, n)
	for i := range dbs {
		dbs[i] = &DatabaseConfig{Name: fmt.Sprintf("db%d", i)}
	}
	return dbs
}

// trackRun returns a run function that records the highest number of calls
// running at the same time.
func trackRun(peak *atomic.Int32) func(*DatabaseConfig) *Result {
	var running atomic.Int32
	return func(db *DatabaseConfig) *Result {
		n := running.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		running.Add(-1)
		return &Result{Name: db.Name}
	}
}

func TestRunPoolRespectsLimitAndKeepsOrder(t *testing.T) {
	for _, tc := range []struct{ n, wantPeak int }{{0, 1}, {1, 1}, {3, 3}, {50, 8}} {
		var peak atomic.Int32
		dbs := namedDBs(8)
		res := runPool(context.Background(), tc.n, dbs, trackRun(&peak))
		if int(peak.Load()) != tc.wantPeak {
			t.Errorf("parallel %d: peak %d, want %d", tc.n, peak.Load(), tc.wantPeak)
		}
		for i, r := range res {
			if r == nil || r.Name != dbs[i].Name {
				t.Fatalf("parallel %d: result %d = %+v, want %s", tc.n, i, r, dbs[i].Name)
			}
		}
	}
}

func TestRunPoolStopsStartingWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var started atomic.Int32
	res := runPool(ctx, 1, namedDBs(5), func(db *DatabaseConfig) *Result {
		if started.Add(1) == 2 {
			cancel()
		}
		return &Result{Name: db.Name}
	})
	done := 0
	for _, r := range res {
		if r != nil {
			done++
		}
	}
	if done < 2 || done >= 5 {
		t.Errorf("%d of 5 databases ran after cancelling during the second", done)
	}
}

func TestRunPoolNoDatabases(t *testing.T) {
	if res := runPool(context.Background(), 4, nil, nil); len(res) != 0 {
		t.Errorf("got %d results for no databases", len(res))
	}
}

func TestGateLimitsConcurrency(t *testing.T) {
	g := newGate(2)
	var running, peak atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if !g.acquire(context.Background()) {
				t.Error("acquire failed")
				return
			}
			defer g.release()
			n := running.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			running.Add(-1)
		})
	}
	wg.Wait()
	if peak.Load() != 2 {
		t.Errorf("peak %d, want 2", peak.Load())
	}
}

func TestGateUnsetAndCancel(t *testing.T) {
	var none gate = newGate(0)
	if none != nil || !none.acquire(context.Background()) {
		t.Error("an unset gate must let everything through")
	}
	none.release() // must not block or panic

	g := newGate(1)
	g.acquire(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if g.acquire(ctx) {
		t.Error("acquire succeeded on a full gate with a cancelled context")
	}
}

func TestParallelConfig(t *testing.T) {
	var c Config
	if err := json.Unmarshal([]byte(`{"parallel":4}`), &c); err != nil || c.Parallel != 4 {
		t.Fatalf("parallel parsed as %d, err %v", c.Parallel, err)
	}
	db := &DatabaseConfig{
		Name:         "shop",
		MysqlConfig:  MysqlConfig{Host: "h", Port: 3306, Username: "u", Database: "d"},
		Destinations: []Destination{{Type: destLocal, Path: "x"}},
	}
	for _, n := range []int{0, 1, 16} {
		if err := (&Config{Parallel: n, Databases: []*DatabaseConfig{db}}).Validate(); err != nil {
			t.Errorf("parallel %d rejected: %v", n, err)
		}
	}
	for _, n := range []int{-1, 17, 1000} {
		if err := (&Config{Parallel: n, Databases: []*DatabaseConfig{db}}).Validate(); err == nil {
			t.Errorf("parallel %d accepted", n)
		}
	}
}
