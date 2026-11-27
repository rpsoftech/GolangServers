package soham_whatsapp_client_apis

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
	whatsapp_client_core "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/core"
)

// stubConnect replaces the connect and JID lookups and the token config for
// one test. The returned channel receives each token passed to connect.
func stubConnect(t *testing.T, block <-chan struct{}, tokens map[string]string) <-chan string {
	t.Helper()
	calls := make(chan string, 8)
	oldConnect, oldJID, oldConfig := connectReserved, jidForToken, whatsapp_config.WhatsappNumberConfigMap
	connectReserved = func(_ string, token string) {
		calls <- token
		<-block
	}
	jidForToken = func(string) string { return "" }
	whatsapp_config.WhatsappNumberConfigMap = &whatsapp_interfaces.IServerConfig{
		Tokens: tokens,
		JID:    map[string]string{},
	}
	t.Cleanup(func() {
		connectReserved, jidForToken, whatsapp_config.WhatsappNumberConfigMap = oldConnect, oldJID, oldConfig
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
	const token = "soham-start-returns-immediately"
	block := make(chan struct{})
	defer close(block)
	calls := stubConnect(t, block, map[string]string{token: "919999999999"})
	t.Cleanup(func() { whatsapp_core.ConnectionMap.Delete(token) })

	// The stubbed connect blocks until the test ends, like a device waiting
	// for a QR scan. app.Test fails after one second if the handler waits.
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

	if number, ok := whatsapp_client_core.ScanNumber(token); !ok || number != "919999999999" {
		t.Fatalf("scan number = %q, %v; want the configured number", number, ok)
	}
}

func TestStartNumberRejectsDuplicate(t *testing.T) {
	const token = "soham-start-rejects-duplicate"
	block := make(chan struct{})
	defer close(block)
	calls := stubConnect(t, block, map[string]string{token: ""})
	t.Cleanup(func() { whatsapp_core.ConnectionMap.Delete(token) })

	app := newStartApp(token)
	if body := postStart(t, app); body["success"] != true {
		t.Fatalf("first call body = %v, want success true", body)
	}
	if body := postStart(t, app); body["success"] != false {
		t.Fatalf("second call body = %v, want success false", body)
	}

	<-calls
	select {
	case got := <-calls:
		t.Fatalf("connect started twice, second token %q", got)
	case <-time.After(100 * time.Millisecond):
	}
}
