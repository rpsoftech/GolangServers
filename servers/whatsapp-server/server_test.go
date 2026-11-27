package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
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

func setupTestServer(t *testing.T) (*lockedBuffer, func(*http.Request) (*http.Response, string)) {
	t.Helper()
	prevEnv, prevConfig := whatsapp_config.Env, whatsapp_config.WhatsappNumberConfigMap
	t.Cleanup(func() {
		whatsapp_config.Env, whatsapp_config.WhatsappNumberConfigMap = prevEnv, prevConfig
	})
	whatsapp_config.Env = &whatsapp_interfaces.EnvInterface{}
	whatsapp_config.WhatsappNumberConfigMap = &whatsapp_interfaces.IServerConfig{
		Tokens: map[string]string{secretToken: ""},
		JID:    map[string]string{},
	}

	logs := &lockedBuffer{}
	app := setupFiberServer(logs)
	do := func(req *http.Request) (*http.Response, string) {
		t.Helper()
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s %s: %v", req.Method, req.URL, err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp, string(body)
	}
	return logs, do
}

func TestV1WithoutTokenIs403(t *testing.T) {
	_, do := setupTestServer(t)
	for _, path := range []string{"/v1/qr_code", "/v1/status", "/v1/send_message", "/v1/anything"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			resp, body := do(httptest.NewRequest(method, path, nil))
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("%s %s: status = %d, want 403 (body %s)", method, path, resp.StatusCode, body)
				continue
			}
			var reqErr interfaces.RequestError
			if err := json.Unmarshal([]byte(body), &reqErr); err != nil || reqErr.Name != "ERROR_TOKEN_NOT_PASSED" {
				t.Errorf("%s %s: body = %s, want ERROR_TOKEN_NOT_PASSED", method, path, body)
			}
		}
	}
}

func TestV1WithUnknownTokenIs401(t *testing.T) {
	_, do := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/qr_code", nil)
	req.Header.Set(env.RequestTokenHeaderKey, "wrong")
	if resp, body := do(req); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", resp.StatusCode, body)
	}
}

func TestNotFoundFallback(t *testing.T) {
	_, do := setupTestServer(t)
	for _, path := range []string{"/", "/nope", "/scan", "/scan/a/b", "/v2/status"} {
		resp, body := do(httptest.NewRequest(http.MethodGet, path, nil))
		if resp.StatusCode != http.StatusNotFound || body != "Sorry can't find that!" {
			t.Errorf("GET %s: status, body = %d, %q; want 404 fallback", path, resp.StatusCode, body)
		}
	}
}

func TestScanPageServed(t *testing.T) {
	_, do := setupTestServer(t)
	resp, body := do(httptest.NewRequest(http.MethodGet, "/scan/"+secretToken, nil))
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `const id = "`+secretToken+`";`) {
		t.Fatalf("status = %d, page misses the id", resp.StatusCode)
	}
}

func TestRequestLogsNeverContainToken(t *testing.T) {
	logs, do := setupTestServer(t)

	paths := []string{
		"/scan/" + secretToken,
		"/SCAN/" + secretToken,
		"/scan/" + secretToken + "/",
		"/scan/" + secretToken + "?x=1",
		"/scan/a/" + secretToken,
	}
	for _, path := range paths {
		do(httptest.NewRequest(http.MethodGet, path, nil))
	}
	for _, path := range []string{"/v1/qr_code", "/v1/status", "/nope"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(env.RequestTokenHeaderKey, secretToken)
		do(req)
	}

	out := logs.String()
	if strings.Count(out, "\n") != len(paths)+3 {
		t.Fatalf("want one log line per request, got:\n%s", out)
	}
	if strings.Contains(out, secretToken) {
		t.Fatalf("request log contains the token:\n%s", out)
	}
	t.Log("\n" + out)
	if !strings.Contains(out, "/scan/***") || !strings.Contains(out, "/v1/qr_code") || !strings.Contains(out, "Number *** Not Found") {
		t.Fatalf("request log misses expected paths:\n%s", out)
	}
}
