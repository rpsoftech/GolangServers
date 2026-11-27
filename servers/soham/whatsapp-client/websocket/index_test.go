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
	soham_whatsapp_client_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/env"
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
	base := filepath.Join(root, "media")
	outside := filepath.Join(root, "outside")
	for _, d := range []string{base, outside, filepath.Join(base, "sub")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p string) {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(base, "a.pdf"))
	write(filepath.Join(base, "sub", "b.pdf"))
	write(filepath.Join(outside, "secret.db"))
	if err := os.Symlink(filepath.Join(outside, "secret.db"), filepath.Join(base, "link-out.pdf")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "a.pdf"), filepath.Join(base, "link-in.pdf")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "dir-out")); err != nil {
		t.Fatal(err)
	}

	realBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		baseDir   string
		requested string
		want      string // resolved path when allowed
		wantErr   string // substring of the error when rejected
	}{
		{"absolute file inside", base, filepath.Join(base, "a.pdf"), filepath.Join(realBase, "a.pdf"), ""},
		{"relative file inside", base, "a.pdf", filepath.Join(realBase, "a.pdf"), ""},
		{"nested file inside", base, filepath.Join("sub", "b.pdf"), filepath.Join(realBase, "sub", "b.pdf"), ""},
		{"symlink staying inside", base, "link-in.pdf", filepath.Join(realBase, "a.pdf"), ""},
		{"dot-dot escape", base, filepath.Join("..", "outside", "secret.db"), "", "outside the allowed directory"},
		{"dot-dot in absolute path", base, filepath.Join(base, "..", "outside", "secret.db"), "", "outside the allowed directory"},
		{"absolute path outside", base, filepath.Join(outside, "secret.db"), "", "outside the allowed directory"},
		{"system file", base, "/etc/hosts", "", "outside the allowed directory"},
		{"symlink pointing out", base, "link-out.pdf", "", "outside the allowed directory"},
		{"file through symlinked dir", base, filepath.Join("dir-out", "secret.db"), "", "outside the allowed directory"},
		{"base directory itself", base, base, "", "outside the allowed directory"},
		{"directory inside", base, "sub", "", "not a regular file"},
		{"missing file", base, "nope.pdf", "", "not found"},
		{"empty path", base, "", "", "empty"},
		{"unset LOCAL_MEDIA_DIR", "", filepath.Join(base, "a.pdf"), "", soham_whatsapp_client_env.LOCAL_MEDIA_DIR_KEY},
		{"missing base directory", filepath.Join(root, "missing"), filepath.Join(base, "a.pdf"), "", "invalid local media directory"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveLocalMediaPath(tc.baseDir, tc.requested)
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

func TestResolveLocalMediaPathPrefixSibling(t *testing.T) {
	// "/x/media" must not allow "/x/media-other/file".
	root := t.TempDir()
	base := filepath.Join(root, "media")
	other := filepath.Join(root, "media-other")
	for _, d := range []string{base, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(other, "f.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveLocalMediaPath(base, filepath.Join(other, "f.pdf")); err == nil {
		t.Fatal("file in sibling directory with the same prefix was allowed")
	}
}

func TestParseLocalMediaDir(t *testing.T) {
	dir := t.TempDir()
	if got, err := soham_whatsapp_client_env.ParseLocalMediaDir(""); err != nil || got != "" {
		t.Fatalf("empty value: got %q, %v", got, err)
	}
	got, err := soham_whatsapp_client_env.ParseLocalMediaDir(dir)
	if err != nil || !filepath.IsAbs(got) {
		t.Fatalf("directory: got %q, %v", got, err)
	}
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := soham_whatsapp_client_env.ParseLocalMediaDir(file); err == nil {
		t.Fatal("file accepted as LOCAL_MEDIA_DIR")
	}
	if _, err := soham_whatsapp_client_env.ParseLocalMediaDir(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing directory accepted as LOCAL_MEDIA_DIR")
	}
}

func TestSendTimeout(t *testing.T) {
	tests := []struct {
		base       time.Duration
		recipients int
		want       time.Duration
	}{
		{textSendTimeout, 0, 10 * time.Second},
		{textSendTimeout, 1, 10 * time.Second},
		{textSendTimeout, 3, 30 * time.Second},
		{mediaSendTimeout, 4, 2 * time.Minute},
		{webMediaSendTimeout, 2, 90 * time.Second},
		{textSendTimeout, 30, maxSendTimeout},
		{mediaSendTimeout, 11, maxSendTimeout},
		{webMediaSendTimeout, 1 << 62, maxSendTimeout},
	}
	for _, tc := range tests {
		if got := sendTimeout(tc.base, tc.recipients); got != tc.want {
			t.Errorf("sendTimeout(%s, %d) = %s, want %s", tc.base, tc.recipients, got, tc.want)
		}
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

func TestFilePathMediaRejectedWhenDirUnset(t *testing.T) {
	setVar(t, &soham_whatsapp_client_env.LocalMediaDir, "")
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
			LocalMediaPath:     "/etc/hosts",
		},
	}
	r := waitReply(t, replies)
	if r.ReqId != "req-file" || r.Message.Message != "Local media path rejected" {
		t.Fatalf("unexpected reply %+v", r)
	}
	if extra, _ := r.Message.Extra.(string); !strings.Contains(extra, soham_whatsapp_client_env.LOCAL_MEDIA_DIR_KEY) {
		t.Fatalf("extra %q does not name %s", extra, soham_whatsapp_client_env.LOCAL_MEDIA_DIR_KEY)
	}
}

func TestHandlersWaitForFreeSlot(t *testing.T) {
	setVar(t, &soham_whatsapp_client_env.LocalMediaDir, "")
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
	if maxMessageSize != 64<<20 {
		t.Fatalf("maxMessageSize = %d", maxMessageSize)
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
