package soham_whatsapp_server_websocket

import (
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	gws "github.com/gorilla/websocket"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
)

func setVar[T any](t *testing.T, p *T, v T) {
	t.Helper()
	old := *p
	*p = v
	t.Cleanup(func() { *p = old })
}

// startServer runs the websocket handler behind a stub of the auth
// middleware and returns its ws:// base URL.
func startServer(t *testing.T) string {
	t.Helper()
	app := fiber.New()
	app.Use("/ws", func(c fiber.Ctx) error {
		c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY, c.Get(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY))
		c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY, "token")
		return c.Next()
	})
	app.Get("/ws", websocket.New(GetWhatsappClientWebsocketHandler().Handle))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() { _ = app.Shutdown() })
	return "ws://" + ln.Addr().String() + "/ws"
}

func dial(t *testing.T, url string, number int) *gws.Conn {
	t.Helper()
	h := http.Header{}
	h.Set(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY, strconv.Itoa(number))
	c, _, err := gws.DefaultDialer.Dial(url, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func registered(number int) bool {
	_, ok := GetWhatsappClientWebsocketHandler().whatsappService.GetStatus(number)
	return ok
}

func TestServerPingsClient(t *testing.T) {
	setVar(t, &pingPeriod, 20*time.Millisecond)
	url := startServer(t)
	c := dial(t, url, 101)
	var pings atomic.Int32
	c.SetPingHandler(func(data string) error {
		pings.Add(1)
		return c.WriteControl(gws.PongMessage, []byte(data), time.Now().Add(time.Second))
	})
	go func() {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}()
	waitFor(t, "server pings", func() bool { return pings.Load() >= 3 })
}

func TestSilentClientIsDropped(t *testing.T) {
	setVar(t, &pingPeriod, time.Hour)
	setVar(t, &pongWait, 150*time.Millisecond)
	url := startServer(t)
	c := dial(t, url, 102)
	waitFor(t, "registration", func() bool { return registered(102) })

	// The client never reads or writes, so the server's read deadline passes.
	waitFor(t, "silent connection removal", func() bool { return !registered(102) })
	_ = c
}

func TestClientPingsKeepConnectionOpen(t *testing.T) {
	setVar(t, &pingPeriod, time.Hour)
	setVar(t, &pongWait, 150*time.Millisecond)
	url := startServer(t)
	c := dial(t, url, 103)
	go func() {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}()
	waitFor(t, "registration", func() bool { return registered(103) })
	for i := 0; i < 10; i++ {
		if err := c.WriteControl(gws.PingMessage, nil, time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !registered(103) {
		t.Fatal("connection dropped although the client kept pinging")
	}
}

func TestReconnectReplacesOldConnection(t *testing.T) {
	setVar(t, &pingPeriod, time.Hour)
	url := startServer(t)
	first := dial(t, url, 104)
	waitFor(t, "first registration", func() bool { return registered(104) })

	second := dial(t, url, 104)
	go func() {
		for {
			if _, _, err := second.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// The server closes the first connection.
	_ = first.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := first.ReadMessage(); err == nil {
		t.Fatal("first connection still open after reconnect")
	}
	// Give the first handler time to run its cleanup.
	time.Sleep(100 * time.Millisecond)
	if !registered(104) {
		t.Fatal("old connection's cleanup removed the new connection")
	}

	// A status message from the new connection is applied.
	if err := second.WriteJSON(soham_common_req_keys.WhatsappClientMessage{
		Type:    soham_common_req_keys.STATUS_MESSAGE,
		Message: string(soham_common_req_keys.LOGGED_IN),
	}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "status update", func() bool {
		st, _ := GetWhatsappClientWebsocketHandler().whatsappService.GetStatus(104)
		return st == soham_common_req_keys.LOGGED_IN
	})
}
