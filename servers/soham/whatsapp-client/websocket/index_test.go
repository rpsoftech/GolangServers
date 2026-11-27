package soham_whatsapp_client_websocket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
)

// setVar sets *p to v for the duration of the test.
func setVar[T any](t *testing.T, p *T, v T) {
	t.Helper()
	old := *p
	*p = v
	t.Cleanup(func() { *p = old })
}

// newWSServer starts a websocket server that runs handle for each connection.
func newWSServer(t *testing.T, handle func(*websocket.Conn)) (*httptest.Server, *url.URL) {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		handle(c)
	}))
	u, err := url.Parse("ws" + strings.TrimPrefix(srv.URL, "http"))
	if err != nil {
		t.Fatal(err)
	}
	return srv, u
}

type reply struct {
	ReqId   string `json:"reqid"`
	Type    string `json:"type"`
	Message struct {
		Message string `json:"message"`
		Extra   any    `json:"extra"`
	} `json:"message"`
}

// echoServer sends each message from send to the client and forwards every
// reply it reads to the returned channel.
func echoServer(t *testing.T, send <-chan any) (*httptest.Server, *url.URL, <-chan reply) {
	t.Helper()
	replies := make(chan reply, 64)
	srv, u := newWSServer(t, func(c *websocket.Conn) {
		if send != nil {
			go func() {
				for m := range send {
					if err := c.WriteJSON(m); err != nil {
						return
					}
				}
			}()
		}
		for {
			var r reply
			if err := c.ReadJSON(&r); err != nil {
				return
			}
			replies <- r
		}
	})
	return srv, u, replies
}

func waitReply(t *testing.T, replies <-chan reply) reply {
	t.Helper()
	select {
	case r := <-replies:
		return r
	case <-time.After(3 * time.Second):
		t.Fatal("no reply from client")
	}
	return reply{}
}

func TestResolveLocalMediaPath(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	other := filepath.Join(root, "anywhere", "deep")
	for _, d := range []string{app, other, filepath.Join(other, "folder")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p string) {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range protectedFiles {
		write(filepath.Join(app, name))
	}
	write(filepath.Join(app, "photo.pdf"))
	write(filepath.Join(other, "invoice.pdf"))
	write(filepath.Join(other, "copy.env"))
	write(filepath.Join(other, ".env"))
	write(filepath.Join(other, ".env.local"))
	write(filepath.Join(other, "data.db"))
	write(filepath.Join(other, "data.sqlite"))
	if err := os.Symlink(filepath.Join(app, "WhatsappSuperSecrete.db"), filepath.Join(other, "innocent.pdf")); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(app, "whatsapp-client.env"), filepath.Join(other, "hardlink.pdf")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(other, "invoice.pdf"), filepath.Join(other, "link.pdf")); err != nil {
		t.Fatal(err)
	}

	real := func(p string) string {
		r, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	tests := []struct {
		name      string
		requested string
		want      string // resolved path when allowed
		wantErr   string // substring of the error when rejected
	}{
		{"file in any directory", filepath.Join(other, "invoice.pdf"), real(filepath.Join(other, "invoice.pdf")), ""},
		{"dot-dot path elsewhere", filepath.Join(other, "folder", "..", "invoice.pdf"), real(filepath.Join(other, "invoice.pdf")), ""},
		{"system file", "/etc/hosts", real("/etc/hosts"), ""},
		{"symlink to normal file", filepath.Join(other, "link.pdf"), real(filepath.Join(other, "invoice.pdf")), ""},
		{"relative file in app dir", "photo.pdf", real(filepath.Join(app, "photo.pdf")), ""},
		{"session database", filepath.Join(app, "WhatsappSuperSecrete.db"), "", "protected"},
		{"database wal", filepath.Join(app, "WhatsappSuperSecrete.db-wal"), "", "protected"},
		{"database relative", "WhatsappSuperSecrete.db", "", "protected"},
		{"database via dot-dot", filepath.Join(other, "..", "..", "app", "WhatsappSuperSecrete.db"), "", "protected"},
		{"env file", filepath.Join(app, "whatsapp-client.env"), "", "protected"},
		{"env file upper case", filepath.Join(app, "WHATSAPP-CLIENT.ENV"), "", "protected"},
		{"server config", filepath.Join(app, "server.config.json"), "", "protected"},
		{"any env file", filepath.Join(other, "copy.env"), "", "protected"},
		{"dot env", filepath.Join(other, ".env"), "", "protected"},
		{"dot env variant", filepath.Join(other, ".env.local"), "", "protected"},
		{"any db file", filepath.Join(other, "data.db"), "", "protected"},
		{"any sqlite file", filepath.Join(other, "data.sqlite"), "", "protected"},
		{"symlink to database", filepath.Join(other, "innocent.pdf"), "", "protected"},
		{"hard link to env", filepath.Join(other, "hardlink.pdf"), "", "protected"},
		{"directory", filepath.Join(other, "folder"), "", "not a regular file"},
		{"missing file", filepath.Join(other, "nope.pdf"), "", "not found"},
		{"empty path", "", "", "empty"},
		{"blank path", "   ", "", "empty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveLocalMediaPath(app, tc.requested)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("got %q, want error containing %q", got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// connectedObject returns an object whose Conn is a live client connection to
// u, as connectAndRead would leave it.
func connectedObject(t *testing.T, u *url.URL) *WebsocketConnectionObject {
	t.Helper()
	con, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { con.Close() })
	return &WebsocketConnectionObject{Url: u, Conn: con, connected: true, NUMBER: "test"}
}

// runConn runs w.connectAndRead in the background. At cleanup it stops the
// connection and waits for it and for every message handler to finish, so
// later tests can change the package variables safely.
func runConn(t *testing.T, w *WebsocketConnectionObject) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		errCh <- w.connectAndRead(ctx)
		close(finished)
	}()
	t.Cleanup(func() {
		cancel()
		<-finished
		slots := w.slots()
		for range maxConcurrentHandlers {
			slots <- struct{}{}
		}
		for range maxConcurrentHandlers {
			<-slots
		}
	})
	return cancel, errCh
}

func TestSendResultNilRepliesWithError(t *testing.T) {
	srv, u, replies := echoServer(t, nil)
	defer srv.Close()
	w := connectedObject(t, u)

	w.sendResult("req-1", nil, "Failed to send base64 media: invalid base64 data")
	r := waitReply(t, replies)
	if r.ReqId != "req-1" || r.Type != string(soham_common_req_keys.REPSONSE_MESSAGE) {
		t.Fatalf("unexpected reply %+v", r)
	}
	if r.Message.Message != "Failed to send base64 media: invalid base64 data" {
		t.Fatalf("reply message = %q", r.Message.Message)
	}

	ok := map[string]bool{"911234567890": true}
	w.sendResult("req-2", &ok, "unused")
	r = waitReply(t, replies)
	if r.ReqId != "req-2" || r.Message.Message != "" {
		t.Fatalf("success reply = %+v, want the result map", r)
	}
}

func TestFilePathMediaRejectsProtectedFile(t *testing.T) {
	send := make(chan any, 1)
	srv, u, replies := echoServer(t, send)
	defer srv.Close()
	defer close(send)

	w := &WebsocketConnectionObject{Url: u, NUMBER: "test"}
	runConn(t, w)

	send <- soham_common_req_keys.WhatsappClientMessage{
		ReqId: "req-file",
		Type:  soham_common_req_keys.SEND_FILE_PATH_MEDIA,
		Message: soham_common_req_keys.SendFilePathMediaType{
			ApiSendRequestBase: soham_common_req_keys.ApiSendRequestBase{From: 1, To: []string{"911234567890"}},
			LocalMediaPath:     filepath.Join(t.TempDir(), "whatsapp-client.env"),
		},
	}
	r := waitReply(t, replies)
	if r.ReqId != "req-file" || r.Message.Message != "Local media path rejected" {
		t.Fatalf("unexpected reply %+v", r)
	}
	if extra, _ := r.Message.Extra.(string); !strings.Contains(extra, "protected") {
		t.Fatalf("extra %q does not say the file is protected", extra)
	}
}

func TestHandlersWaitForFreeSlot(t *testing.T) {
	send := make(chan any, 64)
	srv, u, replies := echoServer(t, send)
	defer srv.Close()
	defer close(send)

	w := &WebsocketConnectionObject{Url: u, NUMBER: "test"}
	// Fill every slot so the next message must wait.
	slots := w.slots()
	for i := 0; i < maxConcurrentHandlers; i++ {
		slots <- struct{}{}
	}

	runConn(t, w)

	const total = maxConcurrentHandlers + 4
	for i := 0; i < total; i++ {
		send <- soham_common_req_keys.WhatsappClientMessage{
			ReqId: "req",
			Type:  soham_common_req_keys.SEND_FILE_PATH_MEDIA,
			Message: soham_common_req_keys.SendFilePathMediaType{
				LocalMediaPath: "x",
			},
		}
	}

	select {
	case r := <-replies:
		t.Fatalf("handler ran while all slots were busy: %+v", r)
	case <-time.After(200 * time.Millisecond):
	}

	// Free the slots; every message must still be handled.
	for i := 0; i < maxConcurrentHandlers; i++ {
		<-slots
	}
	for i := 0; i < total; i++ {
		waitReply(t, replies)
	}
}

func TestKeepAliveSendsPings(t *testing.T) {
	setVar(t, &pingPeriod, 20*time.Millisecond)
	var pings atomic.Int32
	srv, u := newWSServer(t, func(c *websocket.Conn) {
		c.SetPingHandler(func(data string) error {
			pings.Add(1)
			return c.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
		})
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer srv.Close()

	w := &WebsocketConnectionObject{Url: u, NUMBER: "test"}
	cancel, errCh := runConn(t, w)

	deadline := time.Now().Add(3 * time.Second)
	for pings.Load() < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("got %d pings, want at least 3", pings.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-errCh:
	case <-time.After(3 * time.Second):
		t.Fatal("connectAndRead did not return after shutdown")
	}
}

func TestPongKeepsIdleConnectionOpen(t *testing.T) {
	// The server never sends data. Pongs alone must keep the connection
	// alive past several read deadlines.
	setVar(t, &pingPeriod, 20*time.Millisecond)
	setVar(t, &pongWait, 100*time.Millisecond)
	srv, u := newWSServer(t, func(c *websocket.Conn) {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer srv.Close()

	w := &WebsocketConnectionObject{Url: u, NUMBER: "test"}
	_, errCh := runConn(t, w)

	select {
	case err := <-errCh:
		t.Fatalf("idle connection dropped: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
}

func TestReadLimit(t *testing.T) {
	setVar(t, &readLimit, int64(1024))
	srv, u := newWSServer(t, func(c *websocket.Conn) {
		_ = c.WriteMessage(websocket.TextMessage, []byte(strings.Repeat("a", 4096)))
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer srv.Close()

	w := &WebsocketConnectionObject{Url: u, NUMBER: "test"}
	err := w.connectAndRead(context.Background())
	if !errors.Is(err, websocket.ErrReadLimit) {
		t.Fatalf("connectAndRead error = %v, want %v", err, websocket.ErrReadLimit)
	}
}

func TestDefaultReadLimitIsSharedLimit(t *testing.T) {
	if readLimit != soham_common_req_keys.MaxWebsocketMessageSize {
		t.Fatalf("readLimit = %d, want %d", readLimit, soham_common_req_keys.MaxWebsocketMessageSize)
	}
}

func TestConnectionGoroutinesExit(t *testing.T) {
	before := runtime.NumGoroutine()

	srv, u := newWSServer(t, func(c *websocket.Conn) {
		_ = c.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})

	w := &WebsocketConnectionObject{Url: u, NUMBER: "test"}
	// The context is never cancelled: the helper goroutines must end with
	// the connection, not with shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < 5; i++ {
		if err := w.connectAndRead(ctx); err == nil {
			t.Fatal("expected a close error")
		}
	}
	srv.Close()
	http.DefaultClient.CloseIdleConnections()

	deadline := time.Now().Add(3 * time.Second)
	for {
		n := runtime.NumGoroutine()
		if n <= before {
			break
		}
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			buf = buf[:runtime.Stack(buf, true)]
			t.Fatalf("goroutines: %d before, %d after\n%s", before, n, buf)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCheckStatusStopsOnShutdown(t *testing.T) {
	setVar(t, &statusCheckInterval, 10*time.Millisecond)

	t.Run("cancelled later", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		w := &WebsocketConnectionObject{NUMBER: "test"}
		w.mu.Lock()
		w.ctx = ctx
		w.mu.Unlock()

		done := make(chan struct{})
		go func() { w.CheckStatusAndSendResponse(); close(done) }()
		time.Sleep(50 * time.Millisecond)
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("CheckStatusAndSendResponse did not stop")
		}
	})

	t.Run("context set by InitalizeWebsocket after start", func(t *testing.T) {
		w := &WebsocketConnectionObject{NUMBER: "test"}
		done := make(chan struct{})
		// main.go starts both goroutines at the same time.
		go func() { w.CheckStatusAndSendResponse(); close(done) }()
		time.Sleep(30 * time.Millisecond)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		w.InitalizeWebsocket(ctx) // returns at once: ctx is done
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("CheckStatusAndSendResponse did not stop")
		}
	})
}
