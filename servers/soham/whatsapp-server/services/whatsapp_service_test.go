package soham_whatsapp_server_services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	fastws "github.com/fasthttp/websocket"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
)

// fakeConn records writes and reports overlapping WriteMessage calls.
type fakeConn struct {
	writing    atomic.Bool
	overlapped atomic.Bool
	closed     atomic.Bool
	notText    atomic.Bool
	writes     atomic.Int32
	onWrite    func(msg soham_common_req_keys.WhatsappClientMessage)
}

func (f *fakeConn) WriteMessage(messageType int, data []byte) error {
	if !f.writing.CompareAndSwap(false, true) {
		f.overlapped.Store(true)
	}
	if messageType != fastws.TextMessage {
		f.notText.Store(true)
	}
	time.Sleep(time.Millisecond)
	f.writes.Add(1)
	f.writing.Store(false)
	if f.onWrite != nil {
		var msg soham_common_req_keys.WhatsappClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		f.onWrite(msg)
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
		t.Fatal("two WriteMessage calls ran at the same time")
	}
	if fc.notText.Load() {
		t.Fatal("message not written as a text frame")
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

// setVar sets *p to v for the duration of the test.
func setVar[T any](t *testing.T, p *T, v T) {
	t.Helper()
	old := *p
	*p = v
	t.Cleanup(func() { *p = old })
}

func textRequest(to ...string) *soham_common_req_keys.SendTextMessage {
	return &soham_common_req_keys.SendTextMessage{
		ApiSendRequestBase: soham_common_req_keys.ApiSendRequestBase{From: 1, To: to},
		Message:            "hi",
	}
}

func requireRequestError(t *testing.T, err error, status int) {
	t.Helper()
	var reqErr *interfaces.RequestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("error = %v, want *interfaces.RequestError", err)
	}
	if reqErr.StatusCode != status {
		t.Fatalf("status = %d, want %d (%v)", reqErr.StatusCode, status, reqErr)
	}
}

func TestReplyTimeoutCoversClientTimeout(t *testing.T) {
	tests := []struct {
		messageType soham_common_req_keys.MessageType
		recipients  int
	}{
		{soham_common_req_keys.SEND_TEXT_MESSAGE, 1},
		{soham_common_req_keys.SEND_TEXT_MESSAGE, 10},
		{soham_common_req_keys.SEND_BASE64_IMAGE, 4},
		{soham_common_req_keys.SEND_WEB_MEDIA, 3},
		{soham_common_req_keys.SEND_FILE_PATH_MEDIA, 100},
	}
	for _, tc := range tests {
		client := soham_common_req_keys.SendTimeout(tc.messageType, tc.recipients)
		if got, want := replyTimeout(tc.messageType, tc.recipients), client+replyMargin; got != want {
			t.Errorf("replyTimeout(%s, %d) = %s, want %s", tc.messageType, tc.recipients, got, want)
		}
	}
	// Ten text recipients used to time out on the server at a fixed 30s
	// while the client still had 100s.
	if got := replyTimeout(soham_common_req_keys.SEND_TEXT_MESSAGE, 10); got <= 100*time.Second {
		t.Fatalf("replyTimeout for 10 text recipients = %s, want more than 100s", got)
	}
}

func TestSendMessageUsesRecipientTimeout(t *testing.T) {
	var gotType soham_common_req_keys.MessageType
	var gotRecipients int
	setVar(t, &sendTimeout, func(mt soham_common_req_keys.MessageType, n int) time.Duration {
		gotType, gotRecipients = mt, n
		return 20 * time.Millisecond
	})
	setVar(t, &replyMargin, 0)
	s := newService()
	s.AddConnection(1, &soham_whatsapp_server_env.WebsocketConnection{Conn: &fakeConn{}})

	start := time.Now()
	_, err := s.SendTextMessage(context.Background(), textRequest("1", "2", "3"))
	requireRequestError(t, err, http.StatusRequestTimeout)
	if gotType != soham_common_req_keys.SEND_TEXT_MESSAGE || gotRecipients != 3 {
		t.Fatalf("timeout asked for %s with %d recipients, want SEND_TEXT_MESSAGE with 3", gotType, gotRecipients)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("timeout took %s", d)
	}
}

func TestSendMessageCancelledContextReturns499(t *testing.T) {
	s := newService()
	sent := make(chan struct{})
	fc := &fakeConn{onWrite: func(soham_common_req_keys.WhatsappClientMessage) { close(sent) }}
	s.AddConnection(1, &soham_whatsapp_server_env.WebsocketConnection{Conn: fc})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := s.SendTextMessage(ctx, textRequest("1"))
		errCh <- err
	}()
	<-sent
	cancel()
	select {
	case err := <-errCh:
		requireRequestError(t, err, 499)
	case <-time.After(5 * time.Second):
		t.Fatal("sendMessage did not return after cancel")
	}
}

func TestSendMessageParentDeadlineReturns408(t *testing.T) {
	s := newService()
	s.AddConnection(1, &soham_whatsapp_server_env.WebsocketConnection{Conn: &fakeConn{}})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := s.SendTextMessage(ctx, textRequest("1"))
	requireRequestError(t, err, http.StatusRequestTimeout)
}

func TestSendMessageTooLargeReturns413(t *testing.T) {
	setVar(t, &soham_whatsapp_server_env.MaxMessageSize, 1024)
	s := newService()
	fc := &fakeConn{}
	s.AddConnection(1, &soham_whatsapp_server_env.WebsocketConnection{Conn: fc})

	req := textRequest("1")
	req.Message = strings.Repeat("a", 2048)
	_, err := s.SendTextMessage(context.Background(), req)
	requireRequestError(t, err, http.StatusRequestEntityTooLarge)
	if n := fc.writes.Load(); n != 0 {
		t.Fatalf("writes = %d, want 0 for a rejected message", n)
	}
	if len(s.reqIdMap) != 0 {
		t.Fatalf("reqIdMap has %d entries after rejection", len(s.reqIdMap))
	}
}

func TestConnectionSendMessageSizeLimit(t *testing.T) {
	setVar(t, &soham_whatsapp_server_env.MaxMessageSize, 1024)
	fc := &fakeConn{}
	conn := &soham_whatsapp_server_env.WebsocketConnection{Conn: fc}

	err := conn.SendMessage("id", soham_common_req_keys.SEND_TEXT_MESSAGE, strings.Repeat("a", 2048))
	if !errors.Is(err, soham_whatsapp_server_env.ErrMessageTooLarge) {
		t.Fatalf("error = %v, want ErrMessageTooLarge", err)
	}
	if err := conn.SendMessage("id", soham_common_req_keys.SEND_TEXT_MESSAGE, "small"); err != nil {
		t.Fatalf("small message: %v", err)
	}
	if n := fc.writes.Load(); n != 1 {
		t.Fatalf("writes = %d, want 1", n)
	}
}
