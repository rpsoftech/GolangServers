package whatsapp_server_apis

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
)

// stubConnect replaces the connect and JID lookups for one test. The
// returned channel receives each token passed to connect.
func stubConnect(t *testing.T, block <-chan struct{}) <-chan string {
	t.Helper()
	calls := make(chan string, 8)
	oldConnect, oldJID := connectReserved, jidForToken
	connectReserved = func(_ string, token string) {
		calls <- token
		<-block
	}
	jidForToken = func(string) string { return "" }
	t.Cleanup(func() {
		connectReserved, jidForToken = oldConnect, oldJID
	})
	return calls
}

func newStartApp(token string) *fiber.App {
	app := fiber.New()
	app.Post("/v1/start", func(c fiber.Ctx) error {
		c.Locals(whatsapp_interfaces.REQ_LOCAL_NUMBER_KEY, token)
		return c.Next()
	}, StartNumber)
	return app
}

func postStart(t *testing.T, app *fiber.App) map[string]any {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/v1/start", nil), fiber.TestConfig{Timeout: time.Second})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	body := map[string]any{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	return body
}

func TestStartNumberReturnsWhileConnectBlocks(t *testing.T) {
	const token = "start-returns-immediately"
	block := make(chan struct{})
	defer close(block)
	calls := stubConnect(t, block)
	t.Cleanup(func() { whatsapp_core.ConnectionMap.Delete(token) })

	// The stubbed connect never returns until the test ends, like a device
	// waiting for a QR scan. app.Test fails after one second if the handler
	// waits for it.
	body := postStart(t, newStartApp(token))
	if body["success"] != true {
		t.Fatalf("body = %v, want success true", body)
	}

	select {
	case got := <-calls:
		if got != token {
			t.Fatalf("connect token = %q, want %q", got, token)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("connect was not started")
	}
}

func TestStartNumberRejectsDuplicate(t *testing.T) {
	const token = "start-rejects-duplicate"
	block := make(chan struct{})
	defer close(block)
	calls := stubConnect(t, block)
	t.Cleanup(func() { whatsapp_core.ConnectionMap.Delete(token) })

	app := newStartApp(token)
	if body := postStart(t, app); body["success"] != true {
		t.Fatalf("first call body = %v, want success true", body)
	}
	body := postStart(t, app)
	if body["success"] != false {
		t.Fatalf("second call body = %v, want success false", body)
	}

	<-calls
	select {
	case got := <-calls:
		t.Fatalf("connect started twice, second token %q", got)
	case <-time.After(100 * time.Millisecond):
	}
}
