package whatsapp_logging

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
)

const secretToken = "secret-token-value-123"

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestLoggerRedactsScanTokenAndHeader(t *testing.T) {
	logs := &lockedBuffer{}
	app := fiber.New()
	app.Use(NewRedactingLoggerTo(logs))
	app.Get("/scan/:id", func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/v1/fail", func(c fiber.Ctx) error {
		return errors.New("Number " + c.Get(env.RequestTokenHeaderKey) + " Not Found")
	})
	app.Get("/v1/ok", func(c fiber.Ctx) error { return c.SendString("ok") })

	paths := []string{
		"/scan/" + secretToken,
		"/SCAN/" + secretToken,
		"/Scan/" + secretToken + "/",
		"/scan/" + secretToken + "?x=1",
	}
	for _, path := range paths {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	for _, path := range []string{"/v1/ok", "/v1/fail"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(env.RequestTokenHeaderKey, secretToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
	}

	out := logs.String()
	if strings.Count(out, "\n") != len(paths)+2 {
		t.Fatalf("want one log line per request, got:\n%s", out)
	}
	if strings.Contains(out, secretToken) {
		t.Fatalf("request log contains the token:\n%s", out)
	}
	for _, want := range []string{"/scan/***", "/SCAN/***", "/Scan/***/", "/v1/ok", "Number *** Not Found"} {
		if !strings.Contains(out, want) {
			t.Fatalf("request log misses %q:\n%s", want, out)
		}
	}
}

func TestNewRedactingLoggerToNilStream(t *testing.T) {
	if NewRedactingLoggerTo(nil) == nil || NewRedactingLogger() == nil {
		t.Fatal("logger constructor returned nil")
	}
}

func TestRedactLogPath(t *testing.T) {
	cases := map[string]string{
		"/scan/abc":        "/scan/***",
		"/scan/abc/":       "/scan/***/",
		"/Scan/abc":        "/Scan/***",
		"/scan/a/b":        "/scan/***/***",
		"/scan":            "/scan",
		"/scan/":           "/scan/",
		"/v1/qr_code":      "/v1/qr_code",
		"/":                "/",
		"/scanner/abc":     "/scanner/abc",
		"/x/scan/abc":      "/x/scan/***",
		"/v1/send_message": "/v1/send_message",
	}
	for in, want := range cases {
		if got := redactLogPath(in); got != want {
			t.Errorf("redactLogPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRequestLogFormatAvoidsFastPath(t *testing.T) {
	if requestLogFormat == logger.DefaultFormat {
		t.Fatal("requestLogFormat equals logger.DefaultFormat, so CustomTags would be ignored")
	}
	for _, tag := range []string{"${reqHeader", "${reqHeaders}", "${queryParams}", "${query:", "${body}", "${url}", "${cookie"} {
		if strings.Contains(requestLogFormat, tag) {
			t.Fatalf("requestLogFormat contains %s", tag)
		}
	}
}

func TestSanitizeLogValue(t *testing.T) {
	if got := sanitizeLogValue("/a\nb\r\x1b[31m\x7f"); got != "/a?b??[31m?" {
		t.Fatalf("sanitizeLogValue = %q", got)
	}
}
