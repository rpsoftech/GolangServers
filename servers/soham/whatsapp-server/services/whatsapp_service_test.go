package soham_whatsapp_server_services

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
)

// fakeConn records writes and reports overlapping WriteJSON calls.
type fakeConn struct {
	writing    atomic.Bool
	overlapped atomic.Bool
	closed     atomic.Bool
	writes     atomic.Int32
	onWrite    func(msg soham_common_req_keys.WhatsappClientMessage)
}

func (f *fakeConn) WriteJSON(v any) error {
	if !f.writing.CompareAndSwap(false, true) {
		f.overlapped.Store(true)
	}
	time.Sleep(time.Millisecond)
	f.writes.Add(1)
	f.writing.Store(false)
	if f.onWrite != nil {
		f.onWrite(v.(soham_common_req_keys.WhatsappClientMessage))
	}
	return nil
}
func (f *fakeConn) WriteControl(int, []byte, time.Time) error { return nil }
func (f *fakeConn) SetWriteDeadline(time.Time) error          { return nil }
func (f *fakeConn) Close() error                              { f.closed.Store(true); return nil }

func newService() *WhatsappService {
	return &WhatsappService{
		reqIdMap:  make(map[string]chan any),
		connMap:   make(map[int]*soham_whatsapp_server_env.WebsocketConnection),
		statusMap: make(map[int]soham_common_req_keys.ConnectionStatus),
	}
}

func TestSendMessageSerializesWrites(t *testing.T) {
	fc := &fakeConn{}
	conn := &soham_whatsapp_server_env.WebsocketConnection{Conn: fc}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := conn.SendMessage("id", soham_common_req_keys.SEND_TEXT_MESSAGE, "x"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if fc.overlapped.Load() {
		t.Fatal("two WriteJSON calls ran at the same time")
	}
	if got := fc.writes.Load(); got != 50 {
		t.Fatalf("writes = %d, want 50", got)
	}
}

func TestDuplicateReplyDoesNotPanic(t *testing.T) {
	s := newService()
	var reqId string
	fc := &fakeConn{}
	fc.onWrite = func(msg soham_common_req_keys.WhatsappClientMessage) {
		reqId = msg.ReqId
		if !s.RouteResponse(msg.ReqId, "first") {
			t.Error("first reply not delivered")
		}
		if s.RouteResponse(msg.ReqId, "second") {
			t.Error("duplicate reply was delivered")
		}
	}
	s.AddConnection(1, &soham_whatsapp_server_env.WebsocketConnection{Conn: fc})

	res, err := s.SendTextMessage(context.Background(), &soham_common_req_keys.SendTextMessage{
		ApiSendRequestBase: soham_common_req_keys.ApiSendRequestBase{From: 1, To: []string{"1"}},
		Message:            "hi",
	})
	if err != nil || res != "first" {
		t.Fatalf("got %v, %v; want first reply", res, err)
	}
	// A reply after the request returned must be ignored, not panic.
	if s.RouteResponse(reqId, "late") {
		t.Fatal("late reply was delivered")
	}
}

func TestLateReplyAfterCancelDoesNotPanic(t *testing.T) {
	s := newService()
	ids := make(chan string, 1)
	fc := &fakeConn{onWrite: func(msg soham_common_req_keys.WhatsappClientMessage) { ids <- msg.ReqId }}
	s.AddConnection(1, &soham_whatsapp_server_env.WebsocketConnection{Conn: fc})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.SendTextMessage(ctx, &soham_common_req_keys.SendTextMessage{
			ApiSendRequestBase: soham_common_req_keys.ApiSendRequestBase{From: 1, To: []string{"1"}},
			Message:            "hi",
		})
	}()
	id := <-ids
	// Race the reply against the request giving up.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); s.RouteResponse(id, "late") }()
	}
	cancel()
	wg.Wait()
	<-done
	if s.RouteResponse(id, "after") {
		t.Fatal("reply delivered after the request returned")
	}
}

func TestReconnectKeepsNewConnection(t *testing.T) {
	s := newService()
	oldConn := &soham_whatsapp_server_env.WebsocketConnection{Conn: &fakeConn{}}
	newConn := &soham_whatsapp_server_env.WebsocketConnection{Conn: &fakeConn{}}

	s.AddConnection(7, oldConn)
	s.AddConnection(7, newConn)
	if !oldConn.Conn.(*fakeConn).closed.Load() {
		t.Fatal("replaced connection was not closed")
	}

	s.UpdateStatus(7, newConn, soham_common_req_keys.LOGGED_IN)
	// The old handler exits late: its status and removal must be ignored.
	s.UpdateStatus(7, oldConn, soham_common_req_keys.DISCONNECTED)
	s.RemoveConnection(7, oldConn)
	if st, ok := s.GetStatus(7); !ok || st != soham_common_req_keys.LOGGED_IN {
		t.Fatalf("status = %q, %v; want LOGGED_IN", st, ok)
	}

	s.RemoveConnection(7, newConn)
	if _, ok := s.GetStatus(7); ok {
		t.Fatal("connection still registered after its own removal")
	}
}
