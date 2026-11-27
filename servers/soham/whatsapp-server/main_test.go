package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	"github.com/rpsoftech/golang-servers/validator"
)

func newErrorApp(handlerErr error) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler, BodyLimit: 16})
	app.Post("/fail", func(c fiber.Ctx) error { return handlerErr })
	return app
}

func doRequest(t *testing.T, app *fiber.App, method, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("response is not JSON: %q", raw)
	}
	return resp.StatusCode, out
}

func TestErrorHandler(t *testing.T) {
	reqErr := &interfaces.RequestError{StatusCode: 403, Code: 42, Message: "denied", Name: "DENIED"}
	tests := []struct {
		name        string
		err         error
		body        string
		wantStatus  int
		wantCode    float64
		wantMessage string
	}{
		{"request error", reqErr, "", 403, 42, "denied"},
		{"wrapped request error", fmt.Errorf("context: %w", reqErr), "", 403, 42, "denied"},
		{"fiber error", fiber.ErrNotFound, "", 404, 404, fiber.ErrNotFound.Message},
		{"wrapped fiber error", fmt.Errorf("x: %w", fiber.NewError(400, "bad json")), "", 400, 400, "bad json"},
		{"413 fiber error", fiber.ErrRequestEntityTooLarge, "", 413, 413, fiber.ErrRequestEntityTooLarge.Message},
		{"fiber 5xx error", fiber.NewError(500, "secret detail"), "", 500, interfaces.ERROR_INTERNAL_SERVER, "Some Internal Error"},
		{"plain error", errors.New("secret detail"), "", 500, interfaces.ERROR_INTERNAL_SERVER, "Some Internal Error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, out := doRequest(t, newErrorApp(tt.err), "POST", "/fail", tt.body)
			if status != tt.wantStatus {
				t.Fatalf("status %d, want %d (body %v)", status, tt.wantStatus, out)
			}
			if out["code"] != tt.wantCode || out["message"] != tt.wantMessage {
				t.Fatalf("body %v, want code %v message %q", out, tt.wantCode, tt.wantMessage)
			}
			if _, ok := out["extra"]; ok {
				t.Fatalf("extra must not be set: %v", out)
			}
			if strings.Contains(fmt.Sprint(out), "secret detail") {
				t.Fatalf("raw error leaked: %v", out)
			}
		})
	}
}

func TestErrorHandlerUnknownRoute(t *testing.T) {
	app := newErrorApp(nil)
	status, out := doRequest(t, app, "GET", "/missing", "")
	if status != 404 || out["code"] != float64(404) {
		t.Fatalf("got %d %v, want 404", status, out)
	}
}

// TestErrorHandlerBodyTooLarge sends a real request over TCP, because the
// body limit is enforced by the server before any handler runs.
func TestErrorHandlerBodyTooLarge(t *testing.T) {
	app := newErrorApp(nil)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() { _ = app.Shutdown() })
	resp, err := http.Post("http://"+ln.Addr().String()+"/fail", "text/plain", strings.NewReader(strings.Repeat("a", 64)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("response is not JSON: %q", raw)
	}
	if resp.StatusCode != 413 || out["code"] != float64(413) {
		t.Fatalf("got %d %v, want 413", resp.StatusCode, out)
	}
}

// writeSelfSignedCert writes a self-signed certificate and key in PEM form.
func writeSelfSignedCert(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDer, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}

func writeSSLConfig(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, "ssl.config.json")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func sslJSON(cert, key string) string {
	b, _ := json.Marshal(map[string]string{"cert": cert, "key": key})
	return string(b)
}

func TestLoadTLSConfigMissingFile(t *testing.T) {
	cfg, err := loadTLSConfig(filepath.Join(t.TempDir(), "ssl.config.json"))
	if cfg != nil || err != nil {
		t.Fatalf("got %v, %v; want nil, nil", cfg, err)
	}
}

func TestLoadTLSConfigValid(t *testing.T) {
	dir := t.TempDir()
	cert, key := writeSelfSignedCert(t, dir)
	cfg, err := loadTLSConfig(writeSSLConfig(t, dir, sslJSON(cert, key)))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("want 1 certificate, got %d", len(cfg.Certificates))
	}
	if cfg.RootCAs != nil || cfg.ClientCAs != nil {
		t.Fatal("server TLS config must not set RootCAs or ClientCAs")
	}
}

func TestLoadTLSConfigErrors(t *testing.T) {
	dir := t.TempDir()
	cert, key := writeSelfSignedCert(t, dir)
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{"bad json", "{", "parse SSL config"},
		{"missing fields", `{"cert":""}`, "invalid SSL config"},
		{"missing cert file", sslJSON(filepath.Join(dir, "nope.pem"), key), "SSL cert file"},
		{"missing key file", sslJSON(cert, filepath.Join(dir, "nope.pem")), "SSL key file"},
		{"cert and key swapped", sslJSON(key, cert), "load SSL certificate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := t.TempDir()
			cfg, err := loadTLSConfig(writeSSLConfig(t, sub, tt.content))
			if err == nil || cfg != nil {
				t.Fatalf("want error, got cfg %v err %v", cfg, err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestApiSendRequestBaseToRequired(t *testing.T) {
	tests := []struct {
		name  string
		to    []string
		valid bool
	}{
		{"nil", nil, false},
		{"empty", []string{}, false},
		{"one", []string{"919876543210"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := soham_common_req_keys.SendTextMessage{
				ApiSendRequestBase: soham_common_req_keys.ApiSendRequestBase{From: 1, To: tt.to},
				Message:            "hi",
			}
			errs := validator.Validator.Validate(req)
			if got := len(errs) == 0; got != tt.valid {
				t.Fatalf("valid = %v, want %v (errs %v)", got, tt.valid, errs)
			}
		})
	}
}
