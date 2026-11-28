package cli

import (
	"testing"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
	"github.com/rpsoftech/golang-servers/servers/pm2go/daemon"
)

// TestMonitRefreshRealDaemon checks the system RPC and list against a real daemon.
func TestMonitRefreshRealDaemon(t *testing.T) {
	shortHome(t)
	exited := make(chan struct{})
	go func() { daemon.Run(daemon.Options{}); close(exited) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if c, err := defaultDial(); err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("daemon did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Cleanup(func() {
		Main([]string{"kill"})
		select {
		case <-exited:
		case <-time.After(10 * time.Second):
		}
	})
	s := &monitState{selID: -1, noColor: true}
	s.refresh()
	if s.statusErr || !s.haveSys || s.sys.CPUCount < 1 {
		t.Fatalf("refresh: err=%v status=%q sys=%+v", s.statusErr, s.status, s.sys)
	}
	_ = common.MethodSystem
}
