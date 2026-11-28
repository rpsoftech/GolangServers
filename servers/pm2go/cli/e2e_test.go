package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/daemon"
)

// TestEndToEnd runs a real daemon in-process and drives the CLI against it.
func TestEndToEnd(t *testing.T) {
	shortHome(t)
	oldSpawn := spawn
	spawn = func() error { t.Error("daemon must already run"); return nil }
	defer func() { spawn = oldSpawn }()

	exited := make(chan struct{})
	var runErr error
	go func() { runErr = daemon.Run(daemon.Options{}); close(exited) }()
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
	// make sure the daemon goes away even when an assertion fails
	t.Cleanup(func() {
		Main([]string{"kill"})
		select {
		case <-exited:
		case <-time.After(10 * time.Second):
		}
	})

	run := func(args ...string) (int, string, string) {
		var code int
		o, e := capture(t, func() { code = Main(args) })
		return code, o, e
	}

	if c, o, e := run("ping"); c != 0 || !strings.Contains(o, "daemon online, pid") {
		t.Fatalf("ping: %d %q %q", c, o, e)
	}
	if c, o, _ := run("ls"); c != 0 || !strings.Contains(o, "no processes") {
		t.Fatalf("empty ls: %d %q", c, o)
	}

	c, o, e := run("start", "sleep", "--name", "e2e", "--no-autorestart", "--env", "FOO=bar", "--", "300")
	if c != 0 {
		t.Fatalf("start: %d %q %q", c, o, e)
	}
	contains(t, o, "e2e", "online")

	if c, o, _ = run("ls"); c != 0 {
		t.Fatalf("ls: %d", c)
	} else {
		contains(t, o, "e2e", "online")
	}
	if c, o, _ = run("jlist"); c != 0 {
		t.Fatalf("jlist: %d", c)
	} else {
		contains(t, o, `"name": "e2e"`, `"status": "online"`)
	}
	if c, o, _ = run("describe", "e2e"); c != 0 {
		t.Fatalf("describe: %d", c)
	} else {
		contains(t, o, "e2e", "sleep", "300")
	}
	if c, _, _ = run("logs", "e2e", "--nostream"); c != 0 {
		t.Fatalf("logs: %d", c)
	}
	if c, _, _ = run("flush", "e2e"); c != 0 {
		t.Fatalf("flush: %d", c)
	}
	if c, o, _ = run("save"); c != 0 || !strings.Contains(o, "saved 1 process(es)") {
		t.Fatalf("save: %d %q", c, o)
	}

	if c, o, e = run("stop", "e2e"); c != 0 {
		t.Fatalf("stop: %d %q %q", c, o, e)
	}
	contains(t, o, "e2e", "stopped")

	if c, o, _ = run("restart", "e2e"); c != 0 {
		t.Fatalf("restart: %d", c)
	} else {
		contains(t, o, "online")
	}
	if c, _, e = run("scale", "e2e", "2"); c != 0 {
		t.Fatalf("scale: %d %q", c, e)
	}
	if c, o, _ = run("delete", "all"); c != 0 || !strings.Contains(o, "no processes") {
		t.Fatalf("delete: %d %q", c, o)
	}
	if c, _, e = run("stop", "ghost"); c != 1 || !strings.Contains(e, "error:") {
		t.Fatalf("stop unknown: %d %q", c, e)
	}

	// start a process, then kill the daemon: children stop and socket goes away
	if c, _, e = run("start", "sleep 300", "--name", "kid"); c != 0 {
		t.Fatalf("start kid: %d %q", c, e)
	}
	if c, o, _ = run("kill"); c != 0 || !strings.Contains(o, "daemon stopped") {
		t.Fatalf("kill: %d %q", c, o)
	}
	select {
	case <-exited:
		if runErr != nil {
			t.Fatalf("daemon.Run: %v", runErr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not exit")
	}
	if _, err := defaultDial(); err == nil {
		t.Error("socket still accepts connections")
	}
	if c, o, _ = run("kill"); c != 0 || !strings.Contains(o, "daemon is not running") {
		t.Errorf("second kill: %d %q", c, o)
	}
}
