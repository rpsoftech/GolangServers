package whatsapp_core

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestMap() *IWhatsappConnectionMap {
	return &IWhatsappConnectionMap{connections: make(map[string]*WhatsappConnection)}
}

func TestReserveConcurrentHasExactlyOneWinner(t *testing.T) {
	cm := newTestMap()
	const callers = 64
	var wins atomic.Int32
	var start, done sync.WaitGroup
	start.Add(1)
	for i := 0; i < callers; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			if cm.Reserve("token") {
				wins.Add(1)
			}
		}()
	}
	start.Done()
	done.Wait()

	if got := wins.Load(); got != 1 {
		t.Fatalf("Reserve winners = %d, want 1", got)
	}
	connection, ok := cm.Get("token")
	if !ok || connection != nil {
		t.Fatalf("Get after Reserve = (%v, %v), want (nil, true)", connection, ok)
	}
}

func TestReserveRejectsStoredConnection(t *testing.T) {
	cm := newTestMap()
	cm.Set("token", &WhatsappConnection{Token: "token"})
	if cm.Reserve("token") {
		t.Fatal("Reserve succeeded for a token that already has a connection")
	}
}

func TestReleaseOnlyRemovesExpectedEntry(t *testing.T) {
	cm := newTestMap()
	if !cm.Reserve("token") {
		t.Fatal("first Reserve failed")
	}
	other := &WhatsappConnection{Token: "token"}
	if cm.release("token", other) {
		t.Fatal("release removed a reservation it did not own")
	}
	if !cm.release("token", nil) {
		t.Fatal("release did not remove the reservation")
	}
	if !cm.Reserve("token") {
		t.Fatal("Reserve failed after release")
	}

	cm.Set("token", other)
	if cm.release("token", nil) {
		t.Fatal("release of a nil reservation removed a stored connection")
	}
	if !cm.release("token", other) {
		t.Fatal("release did not remove the stored connection")
	}
	if _, ok := cm.Get("token"); ok {
		t.Fatal("token still present after release")
	}
}

func TestSnapshotSkipsReservations(t *testing.T) {
	cm := newTestMap()
	cm.Reserve("pending")
	stored := &WhatsappConnection{Token: "stored"}
	cm.Set("stored", stored)

	got := cm.snapshot()
	if len(got) != 1 || got[0] != stored {
		t.Fatalf("snapshot = %v, want only the stored connection", got)
	}
}

func TestConnectEachRunsTokensInParallel(t *testing.T) {
	cm := newTestMap()
	tokens := []string{"a", "b", "c", "d"}

	var started sync.WaitGroup
	started.Add(len(tokens))
	unblock := make(chan struct{})
	var mu sync.Mutex
	gotJID := map[string]string{}

	wg := cm.ConnectEach(tokens, func(token string) string {
		return "jid-" + token
	}, func(jidString, token string) {
		mu.Lock()
		gotJID[token] = jidString
		mu.Unlock()
		started.Done()
		// Every connect blocks, like a device waiting for a QR scan. If the
		// tokens ran one after another, the others would never start.
		<-unblock
	})

	allStarted := make(chan struct{})
	go func() {
		started.Wait()
		close(allStarted)
	}()
	select {
	case <-allStarted:
	case <-time.After(2 * time.Second):
		close(unblock)
		t.Fatal("connects did not all start while the first one was blocked")
	}
	close(unblock)
	wg.Wait()

	for _, token := range tokens {
		if gotJID[token] != "jid-"+token {
			t.Errorf("token %s got JID %q, want %q", token, gotJID[token], "jid-"+token)
		}
		if _, ok := cm.Get(token); !ok {
			t.Errorf("token %s was not reserved", token)
		}
	}
}

func TestConnectEachSkipsReservedTokens(t *testing.T) {
	cm := newTestMap()
	cm.Reserve("taken")

	var calls atomic.Int32
	wg := cm.ConnectEach([]string{"taken", "free", "free"}, func(string) string { return "" }, func(_, token string) {
		calls.Add(1)
		if token != "free" {
			t.Errorf("connect called for %s", token)
		}
	})
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("connect calls = %d, want 1", got)
	}
}

func TestDisconnectAllIgnoresReservations(t *testing.T) {
	token := "disconnect-all-test"
	if !ConnectionMap.Reserve(token) {
		t.Fatal("Reserve failed")
	}
	defer ConnectionMap.Delete(token)
	// Must not panic on the nil entry a reservation leaves.
	DisconnectAll()
}
