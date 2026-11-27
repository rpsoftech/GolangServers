package whatsapp_middleware

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
)

const (
	validToken     = "valid-token"
	loggedOutToken = "logged-out-token"
)

// setup installs a token config and env for one test and restores the
// previous globals afterwards.
func setup(t *testing.T, allowLocalNoAuth bool, tokens map[string]string) {
	t.Helper()
	prevEnv, prevConfig := whatsapp_config.Env, whatsapp_config.WhatsappNumberConfigMap
	t.Cleanup(func() {
		whatsapp_config.Env, whatsapp_config.WhatsappNumberConfigMap = prevEnv, prevConfig
	})
	whatsapp_config.Env = &whatsapp_interfaces.EnvInterface{ALLOW_LOCAL_NO_AUTH: allowLocalNoAuth}
	whatsapp_config.WhatsappNumberConfigMap = &whatsapp_interfaces.IServerConfig{
		Tokens: tokens,
		JID:    map[string]string{},
	}
}

func defaultTokens() map[string]string {
	return map[string]string{
		validToken:     "jid@s.whatsapp.net",
		loggedOutToken: "",
		"default":      "default-jid@s.whatsapp.net",
	}
}

// newApp builds an app like the server: TokenDecrypter and
// AllowOnlyValidTokenMiddleWare on a group, and an error handler that writes
// RequestError status codes. X-Forwarded-For sets the client IP; the test
// connection itself is 0.0.0.0, which is not loopback.
func newApp(extra ...fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{
		ProxyHeader: fiber.HeaderXForwardedFor,
		TrustProxy:  true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: []string{"0.0.0.0"},
		},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if reqErr, ok := err.(*interfaces.RequestError); ok {
				return c.Status(reqErr.StatusCode).JSON(reqErr)
			}
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		},
	})
	handlers := append([]any{TokenDecrypter, AllowOnlyValidTokenMiddleWare}, toAny(extra)...)
	group := app.Group("/v1", handlers...)
	group.Get("/whoami", func(c fiber.Ctx) error {
		return c.SendString(c.Locals(whatsapp_interfaces.REQ_LOCAL_NUMBER_KEY).(string))
	})
	return app
}

func toAny(handlers []fiber.Handler) []any {
	out := make([]any, len(handlers))
	for i, h := range handlers {
		out[i] = h
	}
	return out
}

type result struct {
	status int
	body   string
	err    interfaces.RequestError
}

func do(t *testing.T, app *fiber.App, token, clientIP string, setToken bool) result {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	if setToken {
		req.Header.Set(env.RequestTokenHeaderKey, token)
	}
	if clientIP != "" {
		req.Header.Set(fiber.HeaderXForwardedFor, clientIP)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	r := result{status: resp.StatusCode, body: string(body)}
	if resp.StatusCode != http.StatusOK {
		_ = json.Unmarshal(body, &r.err)
	}
	return r
}

func TestMissingTokenIs403(t *testing.T) {
	setup(t, false, defaultTokens())
	app := newApp()
	for _, tc := range []struct {
		name     string
		token    string
		setToken bool
	}{
		{"no header", "", false},
		{"empty header", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := do(t, app, tc.token, "203.0.113.7", tc.setToken)
			if r.status != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body %s)", r.status, r.body)
			}
			if r.err.Name != "ERROR_TOKEN_NOT_PASSED" {
				t.Fatalf("error name = %q, want ERROR_TOKEN_NOT_PASSED", r.err.Name)
			}
		})
	}
}

func TestUnknownTokenIs401(t *testing.T) {
	setup(t, false, defaultTokens())
	r := do(t, newApp(), "not-a-token", "203.0.113.7", true)
	if r.status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", r.status, r.body)
	}
	if r.err.Name != "ERROR_INVALID_TOKEN" {
		t.Fatalf("error name = %q, want ERROR_INVALID_TOKEN", r.err.Name)
	}
}

func TestValidTokenPasses(t *testing.T) {
	setup(t, false, defaultTokens())
	r := do(t, newApp(), validToken, "203.0.113.7", true)
	if r.status != http.StatusOK || r.body != validToken {
		t.Fatalf("status, body = %d, %q; want 200, %q", r.status, r.body, validToken)
	}
}

func TestValidTokenWinsOverNoAuth(t *testing.T) {
	setup(t, true, defaultTokens())
	r := do(t, newApp(), validToken, "127.0.0.1", true)
	if r.status != http.StatusOK || r.body != validToken {
		t.Fatalf("status, body = %d, %q; want 200, %q", r.status, r.body, validToken)
	}
}

func TestNoAuthOnlyWhenEnabledAndLoopback(t *testing.T) {
	cases := []struct {
		name     string
		enabled  bool
		clientIP string
		want     int
	}{
		{"enabled, IPv4 loopback", true, "127.0.0.1", http.StatusOK},
		{"enabled, other IPv4 loopback", true, "127.0.0.2", http.StatusOK},
		{"enabled, IPv6 loopback", true, "::1", http.StatusOK},
		{"enabled, public IP", true, "203.0.113.7", http.StatusForbidden},
		{"enabled, private IP", true, "192.168.1.20", http.StatusForbidden},
		{"enabled, link-local IP", true, "169.254.169.254", http.StatusForbidden},
		{"enabled, unspecified IP", true, "0.0.0.0", http.StatusForbidden},
		{"disabled, loopback", false, "127.0.0.1", http.StatusForbidden},
		{"disabled, public IP", false, "203.0.113.7", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setup(t, tc.enabled, defaultTokens())
			r := do(t, newApp(), "", tc.clientIP, false)
			if r.status != tc.want {
				t.Fatalf("status = %d, want %d (body %s)", r.status, tc.want, r.body)
			}
			if tc.want == http.StatusOK && r.body != "default" {
				t.Fatalf("body = %q, want default", r.body)
			}
		})
	}
}

// No-auth maps to the "default" token; without it in the config the request
// still fails, as an unknown token.
func TestNoAuthNeedsDefaultToken(t *testing.T) {
	setup(t, true, map[string]string{validToken: "jid"})
	r := do(t, newApp(), "", "127.0.0.1", false)
	if r.status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", r.status)
	}
}

// The test connection address is 0.0.0.0, so use a real loopback listener
// to check the no-auth path without proxy headers, and check that a spoofed
// header cannot change the IP when no proxy is trusted.
func TestNoAuthRealLoopbackConnection(t *testing.T) {
	setup(t, true, defaultTokens())
	app := fiber.New()
	app.Group("/v1", TokenDecrypter, AllowOnlyValidTokenMiddleWare).Get("/whoami", func(c fiber.Ctx) error {
		return c.SendString(c.Locals(whatsapp_interfaces.REQ_LOCAL_NUMBER_KEY).(string))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	t.Cleanup(func() { _ = app.Shutdown() })

	client := &http.Client{Timeout: 5 * time.Second}
	var resp *http.Response
	for i := 0; i < 50; i++ {
		resp, err = client.Get("http://" + ln.Addr().String() + "/v1/whoami")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "default" {
		t.Fatalf("status, body = %d, %q; want 200, default", resp.StatusCode, body)
	}
}

func TestSpoofedForwardedForIgnoredWithoutTrustedProxy(t *testing.T) {
	setup(t, true, defaultTokens())
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if reqErr, ok := err.(*interfaces.RequestError); ok {
				return c.Status(reqErr.StatusCode).JSON(reqErr)
			}
			return err
		},
	})
	app.Group("/v1", TokenDecrypter, AllowOnlyValidTokenMiddleWare).Get("/whoami", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req.Header.Set(fiber.HeaderXForwardedFor, "127.0.0.1")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestAllowOnlyValidLoggedInWhatsapp(t *testing.T) {
	setup(t, false, defaultTokens())
	app := newApp(AllowOnlyValidLoggedInWhatsapp)

	if r := do(t, app, validToken, "203.0.113.7", true); r.status != http.StatusOK {
		t.Fatalf("logged-in token: status = %d, want 200", r.status)
	}
	r := do(t, app, loggedOutToken, "203.0.113.7", true)
	if r.status != http.StatusUnauthorized || r.err.Name != "ERROR_CONNECTION_LOGGED_OUT" {
		t.Fatalf("logged-out token: status, name = %d, %q; want 401, ERROR_CONNECTION_LOGGED_OUT", r.status, r.err.Name)
	}
}
