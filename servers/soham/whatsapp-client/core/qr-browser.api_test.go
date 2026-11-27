package whatsapp_client_core

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func newScanApp() *fiber.App {
	app := fiber.New(fiber.Config{UnescapePath: true})
	app.Get("/scan/:id", OpenBrowserWithQr)
	return app
}

func getScan(t *testing.T, app *fiber.App, rawPath string) (int, string) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, rawPath, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", rawPath, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestScanUnknownIDIs404(t *testing.T) {
	status, body := getScan(t, newScanApp(), "/scan/not-registered-id")
	if status != http.StatusNotFound || body != "Invalid ID" {
		t.Fatalf("status, body = %d, %q; want 404 Invalid ID", status, body)
	}
}

func TestScanEscapesNumberAndID(t *testing.T) {
	const id = `a"<b>&'`
	SetScanNumber(id, `<i>"9112"</i>`)

	status, body := getScan(t, newScanApp(), "/scan/a%22%3Cb%3E%26'")
	if status != http.StatusOK {
		t.Fatalf("status = %d, body %q", status, body)
	}
	if !strings.Contains(body, "<h2>Number: &lt;i&gt;&#34;9112&#34;&lt;/i&gt;</h2>") {
		t.Fatalf("number not HTML-escaped:\n%s", body)
	}
	// json.Marshal writes <, > and & as <, > and &.
	if !strings.Contains(body, "const id = \"a\\\"\\u003cb\\u003e\\u0026'\";") {
		t.Fatalf("id not JS-escaped:\n%s", body)
	}
	if strings.Contains(body, "<b>") || strings.Contains(body, "<i>") {
		t.Fatalf("raw markup in page:\n%s", body)
	}
}

func TestScanNumberAccessorConcurrent(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 200 {
			SetScanNumber("concurrent-token", "1")
		}
	}()
	for range 200 {
		ScanNumber("concurrent-token")
	}
	<-done
	if n, ok := ScanNumber("concurrent-token"); !ok || n != "1" {
		t.Fatalf("ScanNumber = %q, %v", n, ok)
	}
}
