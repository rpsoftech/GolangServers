package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
	"github.com/rpsoftech/golang-servers/servers/pm2go/daemon"
)

func TestOfflineEventsAreBoundedAndReplayedAfterHello(t *testing.T) {
	t.Setenv("PMGO_HOME", t.TempDir())
	a := New(daemon.NewManager(), "test")

	// Offline: more events than the bound; only the newest survive, in order.
	for i := 0; i < maxPendingEvents+20; i++ {
		a.onEvent(common.Event{TS: int64(i + 1), Name: "api", Type: "exit", Reason: "crash"})
	}
	a.mu.Lock()
	n, first := len(a.pending), a.pending[0].TS
	a.mu.Unlock()
	if n != maxPendingEvents || first != 21 {
		t.Fatalf("pending = %d events, first ts %d; want %d and 21", n, first, maxPendingEvents)
	}

	got := make(chan common.Envelope, 1024)
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			var env common.Envelope
			if c.ReadJSON(&env) != nil {
				return
			}
			got <- env
		}
	}))
	defer srv.Close()
	if err := common.SaveLink(common.LinkConfig{HubURL: srv.URL, Key: "k", Machine: "m1"}); err != nil {
		t.Fatal(err)
	}
	a.Reload()
	defer a.Stop()

	next := func() common.Envelope {
		select {
		case e := <-got:
			return e
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for agent message")
			return common.Envelope{}
		}
	}
	if first := next(); first.Type != common.MsgHello {
		t.Fatalf("first message = %q, want hello", first.Type)
	}
	var lastTS int64
	events := 0
	for events < maxPendingEvents {
		env := next()
		if env.Type != common.MsgEvent {
			continue // snapshots may interleave
		}
		var e common.Event
		if err := json.Unmarshal(env.Data, &e); err != nil {
			t.Fatal(err)
		}
		if e.TS <= lastTS {
			t.Fatalf("replay out of order: %d after %d", e.TS, lastTS)
		}
		lastTS = e.TS
		events++
	}
	if lastTS != int64(maxPendingEvents+20) {
		t.Fatalf("last replayed ts = %d", lastTS)
	}
	a.mu.Lock()
	left := len(a.pending)
	a.mu.Unlock()
	if left != 0 {
		t.Fatalf("%d events still pending after replay", left)
	}
}
