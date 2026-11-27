package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
)

func TestPrepareTmpDirClearsAndCreates(t *testing.T) {
	base := t.TempDir()
	stale := filepath.Join(base, "tmp", "old", "stale.bin")
	if err := os.MkdirAll(filepath.Dir(stale), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	dir, err := prepareTmpDir(base)
	if err != nil {
		t.Fatalf("prepareTmpDir: %v", err)
	}
	if dir != filepath.Join(base, "tmp") {
		t.Fatalf("dir = %q, want %q", dir, filepath.Join(base, "tmp"))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("tmp dir missing: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("tmp dir not empty: %v", entries)
	}
}

func TestPrepareTmpDirCreatesWhenMissing(t *testing.T) {
	base := t.TempDir()
	dir, err := prepareTmpDir(base)
	if err != nil {
		t.Fatalf("prepareTmpDir: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("tmp dir not created: %v", err)
	}
}

func decodeRequestError(t *testing.T, resp *http.Response) interfaces.RequestError {
	t.Helper()
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var reqErr interfaces.RequestError
	if err := json.Unmarshal(raw, &reqErr); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	return reqErr
}

func TestErrorHandlerUnwrapsRequestError(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler})
	app.Get("/wrapped", func(c fiber.Ctx) error {
		return fmt.Errorf("context: %w", &interfaces.RequestError{
			StatusCode: http.StatusNotFound,
			Code:       interfaces.ERROR_CONNECTION_NOT_FOUND,
			Message:    "not here",
			Name:       "ERROR_CONNECTION_NOT_FOUND",
		})
	})
	app.Get("/plain", func(c fiber.Ctx) error {
		return fmt.Errorf("boom")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/wrapped", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("wrapped status = %d, want 404", resp.StatusCode)
	}
	if got := decodeRequestError(t, resp); got.Name != "ERROR_CONNECTION_NOT_FOUND" {
		t.Fatalf("wrapped body name = %q", got.Name)
	}

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/plain", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("plain status = %d, want 500", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestV1WithoutTokenIs403(t *testing.T) {
	prevEnv, prevConfig := whatsapp_config.Env, whatsapp_config.WhatsappNumberConfigMap
	t.Cleanup(func() {
		whatsapp_config.Env, whatsapp_config.WhatsappNumberConfigMap = prevEnv, prevConfig
	})
	whatsapp_config.Env = &whatsapp_interfaces.EnvInterface{}
	whatsapp_config.WhatsappNumberConfigMap = &whatsapp_interfaces.IServerConfig{
		Tokens: map[string]string{"some-token": "911234567890"},
		JID:    map[string]string{},
	}

	app := InitFiberServer()
	routes := []struct{ method, path string }{
		{http.MethodGet, "/v1/qr_code"},
		{http.MethodGet, "/v1/status"},
		{http.MethodPost, "/v1/start"},
		{http.MethodGet, "/v1/anything"},
	}
	for _, r := range routes {
		resp, err := app.Test(httptest.NewRequest(r.method, r.path, nil))
		if err != nil {
			t.Fatalf("%s %s: %v", r.method, r.path, err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: status = %d, want 403", r.method, r.path, resp.StatusCode)
			resp.Body.Close()
			continue
		}
		if got := decodeRequestError(t, resp); got.Name != "ERROR_TOKEN_NOT_PASSED" {
			t.Errorf("%s %s: name = %q, want ERROR_TOKEN_NOT_PASSED", r.method, r.path, got.Name)
		}
	}
}
