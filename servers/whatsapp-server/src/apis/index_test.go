package whatsapp_server_apis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
)

const testToken = "test-token"

// sendCall records one call to the stub connection.
type sendCall struct {
	method   string
	to       []string
	msg      string
	fileName string
	path     string
	content  []byte // file content at the time of the send
	urls     []string
	timeout  time.Duration
	hasLimit bool
}

type stubSender struct {
	statusErr error
	nilResult bool // media sends return nil, as core does on a prepare failure
	release   chan struct{}
	calls     chan sendCall
}

func newStub() *stubSender {
	return &stubSender{calls: make(chan sendCall, 16)}
}

func (s *stubSender) record(ctx context.Context, call sendCall) {
	if deadline, ok := ctx.Deadline(); ok {
		call.hasLimit = true
		call.timeout = time.Until(deadline)
	}
	if call.path != "" {
		call.content, _ = os.ReadFile(call.path)
	}
	if s.release != nil {
		<-s.release
	}
	s.calls <- call
}

func (s *stubSender) result(to []string) *map[string]bool {
	if s.nilResult {
		return nil
	}
	out := make(map[string]bool, len(to))
	for _, n := range to {
		out[n] = true
	}
	return &out
}

func (s *stubSender) ReturnStatusError() error { return s.statusErr }

func (s *stubSender) SendTextMessage(ctx context.Context, to []string, msg string) *map[string]bool {
	s.record(ctx, sendCall{method: "text", to: to, msg: msg})
	out := make(map[string]bool, len(to))
	for _, n := range to {
		out[n] = true
	}
	return &out
}

func (s *stubSender) SendMediaFileFromURLs(ctx context.Context, to []string, urls []string, msg string) {
	s.record(ctx, sendCall{method: "urls", to: to, urls: urls, msg: msg})
}

func (s *stubSender) SendMediaFileFromURL(ctx context.Context, to []string, url string, fileName string, msg string) *map[string]bool {
	s.record(ctx, sendCall{method: "url", to: to, urls: []string{url}, fileName: fileName, msg: msg})
	return s.result(to)
}

func (s *stubSender) SendMediaFileBase64(ctx context.Context, to []string, data string, fileName string, msg string) *map[string]bool {
	s.record(ctx, sendCall{method: "base64", to: to, fileName: fileName, msg: msg})
	return s.result(to)
}

func (s *stubSender) SendMediaFileWithPath(ctx context.Context, to []string, path string, fileName string, msg string) *map[string]bool {
	s.record(ctx, sendCall{method: "path", to: to, path: path, fileName: fileName, msg: msg})
	return s.result(to)
}

// newTestApp wires the handlers with the production error handler shape, a
// fixed token in Locals, the stub connection and a temporary upload dir.
func newTestApp(t *testing.T, stub *stubSender) *fiber.App {
	t.Helper()
	oldGet, oldDir := getConnection, uploadDir
	getConnection = func(token string) (whatsappSender, bool) {
		if token != testToken || stub == nil {
			return nil, false
		}
		return stub, true
	}
	uploadDir = t.TempDir()
	t.Cleanup(func() { getConnection, uploadDir = oldGet, oldDir })

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			var mapped *interfaces.RequestError
			if !errors.As(err, &mapped) {
				return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
			}
			return c.Status(mapped.StatusCode).JSON(mapped)
		},
	})
	app.Use(func(c fiber.Ctx) error {
		c.Locals(whatsapp_interfaces.REQ_LOCAL_NUMBER_KEY, testToken)
		return c.Next()
	})
	app.Post("/send_message", SendMessage)
	app.Post("/send_media", SendMediaFile)
	app.Post("/send_media_64", SendMediaFileWithBase64)
	app.Post("/send_media_url", SendMediaFileWithWebLink)
	app.Post("/send_media_url_multiple", SendMediaFileWithWebLinks)
	return app
}

type testResponse struct {
	status int
	body   []byte
}

func (r testResponse) errorCode(t *testing.T) int {
	t.Helper()
	var e struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(r.body, &e); err != nil {
		t.Fatalf("response is not an error object: %s", r.body)
	}
	return e.Code
}

func do(t *testing.T, app *fiber.App, req *http.Request) testResponse {
	t.Helper()
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return testResponse{status: resp.StatusCode, body: body}
}

func jsonRequest(path, body string, headless bool) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if headless {
		req.Header.Set("Headless", "true")
	}
	return req
}

func multipartRequest(t *testing.T, fields map[string]string, fileName string, content []byte, headless bool) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if fileName != "" {
		fw, err := w.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(content)
	}
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/send_media", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if headless {
		req.Header.Set("Headless", "true")
	}
	return req
}

func waitCall(t *testing.T, stub *stubSender) sendCall {
	t.Helper()
	select {
	case call := <-stub.calls:
		return call
	case <-time.After(5 * time.Second):
		t.Fatal("send was not called")
		return sendCall{}
	}
}

func assertNoCall(t *testing.T, stub *stubSender) {
	t.Helper()
	select {
	case call := <-stub.calls:
		t.Fatalf("unexpected send: %+v", call)
	default:
	}
}

func waitRemoved(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("upload %s was not removed", path)
}

func assertUploadDirEmpty(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("upload dir not empty: %v", entries)
	}
}

func assertTimeout(t *testing.T, call sendCall, want time.Duration) {
	t.Helper()
	if !call.hasLimit {
		t.Fatalf("%s send has no deadline", call.method)
	}
	if call.timeout > want || call.timeout < want-5*time.Second {
		t.Fatalf("%s send timeout = %v, want about %v", call.method, call.timeout, want)
	}
}

// --- bind errors (item 3) ---

func TestJSONHandlersRejectInvalidBody(t *testing.T) {
	for _, path := range []string{"/send_message", "/send_media_64", "/send_media_url", "/send_media_url_multiple"} {
		t.Run(path, func(t *testing.T) {
			stub := newStub()
			app := newTestApp(t, stub)
			resp := do(t, app, jsonRequest(path, `{"to": ["919876543210"], "msg": `, false))
			if resp.status != http.StatusBadRequest || resp.errorCode(t) != interfaces.ERROR_INVALID_INPUT {
				t.Fatalf("got %d %s, want 400 ERROR_INVALID_INPUT", resp.status, resp.body)
			}
			assertNoCall(t, stub)
		})
	}
}

func TestSendMessageRejectsWrongFieldType(t *testing.T) {
	stub := newStub()
	app := newTestApp(t, stub)
	resp := do(t, app, jsonRequest("/send_message", `{"to": "919876543210", "msg": "hi"}`, false))
	if resp.status != http.StatusBadRequest || resp.errorCode(t) != interfaces.ERROR_INVALID_INPUT {
		t.Fatalf("got %d %s, want 400", resp.status, resp.body)
	}
	assertNoCall(t, stub)
}

// --- recipient validation (item 6) ---

func TestRecipientValidation(t *testing.T) {
	cases := []struct {
		name string
		to   string
		ok   bool
		want []string
	}{
		{"valid", `["919876543210"]`, true, []string{"919876543210"}},
		{"leading plus stripped", `["+919876543210", " 9876543 "]`, true, []string{"919876543210", "9876543"}},
		{"second entry too short", `["919876543210", "12345"]`, false, nil},
		{"second entry not digits", `["919876543210", "91987abc210"]`, false, nil},
		{"plus only", `["+"]`, false, nil},
		{"inner plus", `["91+9876543210"]`, false, nil},
		{"empty list", `[]`, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := newStub()
			app := newTestApp(t, stub)
			resp := do(t, app, jsonRequest("/send_message", `{"to": `+tc.to+`, "msg": "hi"}`, false))
			if !tc.ok {
				if resp.status != http.StatusBadRequest {
					t.Fatalf("got %d %s, want 400", resp.status, resp.body)
				}
				assertNoCall(t, stub)
				return
			}
			if resp.status != http.StatusOK {
				t.Fatalf("got %d %s, want 200", resp.status, resp.body)
			}
			call := waitCall(t, stub)
			if strings.Join(call.to, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("to = %v, want %v", call.to, tc.want)
			}
		})
	}
}

func TestMediaHandlersValidateEveryRecipient(t *testing.T) {
	bodies := map[string]string{
		"/send_media_64":           `{"to": ["919876543210", "bad"], "fileName": "a.pdf", "base64": "aGVsbG8="}`,
		"/send_media_url":          `{"to": ["919876543210", "bad"], "url": "https://example.com/a.pdf"}`,
		"/send_media_url_multiple": `{"to": ["919876543210", "bad"], "urls": ["https://example.com/a.pdf"]}`,
	}
	for path, body := range bodies {
		t.Run(path, func(t *testing.T) {
			stub := newStub()
			app := newTestApp(t, stub)
			resp := do(t, app, jsonRequest(path, body, false))
			if resp.status != http.StatusBadRequest {
				t.Fatalf("got %d %s, want 400", resp.status, resp.body)
			}
			assertNoCall(t, stub)
		})
	}
}

func TestConnectionNotFound(t *testing.T) {
	app := newTestApp(t, nil)
	resp := do(t, app, jsonRequest("/send_message", `{"to": ["919876543210"], "msg": "hi"}`, false))
	if resp.status != http.StatusNotFound || resp.errorCode(t) != interfaces.ERROR_CONNECTION_NOT_FOUND {
		t.Fatalf("got %d %s, want 404", resp.status, resp.body)
	}
}

func TestConnectionStatusError(t *testing.T) {
	stub := newStub()
	stub.statusErr = &interfaces.RequestError{StatusCode: http.StatusNotFound, Code: interfaces.ERROR_CONNECTION_LOGGED_OUT}
	app := newTestApp(t, stub)
	resp := do(t, app, jsonRequest("/send_media_64", `{"to": ["919876543210"], "fileName": "a.pdf", "base64": "aGVsbG8="}`, false))
	if resp.status != http.StatusNotFound || resp.errorCode(t) != interfaces.ERROR_CONNECTION_LOGGED_OUT {
		t.Fatalf("got %d %s, want 404 logged out", resp.status, resp.body)
	}
	assertNoCall(t, stub)
}

// --- multipart fields (items 2 and 3) ---

func TestSendMediaFileFields(t *testing.T) {
	cases := []struct {
		name    string
		to, msg string
		wantTo  []string
		wantMsg string
	}{
		{"plain msg and JSON to", `["919876543210","+919876543211"]`, "hello there", []string{"919876543210", "919876543211"}, "hello there"},
		{"comma separated to", "919876543210, 919876543211", "hi", []string{"919876543210", "919876543211"}, "hi"},
		{"single number to", "919876543210", "", []string{"919876543210"}, ""},
		{"JSON number array", `[919876543210]`, "x", []string{"919876543210"}, "x"},
		{"old JSON-quoted msg", `["919876543210"]`, `"quoted \"caption\""`, []string{"919876543210"}, `quoted "caption"`},
		{"msg with brackets", `["919876543210"]`, `[not json`, []string{"919876543210"}, `[not json`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := newStub()
			app := newTestApp(t, stub)
			req := multipartRequest(t, map[string]string{"to": tc.to, "msg": tc.msg}, "report.pdf", []byte("pdf"), false)
			resp := do(t, app, req)
			if resp.status != http.StatusOK {
				t.Fatalf("got %d %s, want 200", resp.status, resp.body)
			}
			call := waitCall(t, stub)
			if strings.Join(call.to, ",") != strings.Join(tc.wantTo, ",") {
				t.Fatalf("to = %v, want %v", call.to, tc.wantTo)
			}
			if call.msg != tc.wantMsg {
				t.Fatalf("msg = %q, want %q", call.msg, tc.wantMsg)
			}
		})
	}
}

func TestSendMediaFileRejectsBadInput(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]string
		file   string
		status int
	}{
		{"unparseable to", map[string]string{"to": `["919876543210",`}, "a.pdf", http.StatusBadRequest},
		{"to array of objects", map[string]string{"to": `[{"n":1}]`}, "a.pdf", http.StatusBadRequest},
		{"missing to", map[string]string{"msg": "hi"}, "a.pdf", http.StatusBadRequest},
		{"bad second number", map[string]string{"to": "919876543210,12"}, "a.pdf", http.StatusBadRequest},
		{"missing file", map[string]string{"to": "919876543210"}, "", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := newStub()
			app := newTestApp(t, stub)
			resp := do(t, app, multipartRequest(t, tc.fields, tc.file, []byte("x"), false))
			if resp.status != tc.status || resp.errorCode(t) != interfaces.ERROR_INVALID_INPUT {
				t.Fatalf("got %d %s, want %d", resp.status, resp.body, tc.status)
			}
			assertNoCall(t, stub)
			assertUploadDirEmpty(t)
		})
	}
}

func TestSendMediaFileRejectsJSONBody(t *testing.T) {
	stub := newStub()
	app := newTestApp(t, stub)
	resp := do(t, app, jsonRequest("/send_media", `{"to": ["919876543210"]}`, false))
	if resp.status != http.StatusBadRequest {
		t.Fatalf("got %d %s, want 400", resp.status, resp.body)
	}
	assertNoCall(t, stub)
}

// --- upload files (item 1) ---

func TestSendMediaFileUniqueUploadRemovedAfterSend(t *testing.T) {
	stub := newStub()
	app := newTestApp(t, stub)
	req := multipartRequest(t, map[string]string{"to": "919876543210"}, "invoice.pdf", []byte("first"), false)
	resp := do(t, app, req)
	if resp.status != http.StatusOK {
		t.Fatalf("got %d %s, want 200", resp.status, resp.body)
	}
	call := waitCall(t, stub)
	if call.fileName != "invoice.pdf" {
		t.Fatalf("fileName = %q, want original name", call.fileName)
	}
	if filepath.Base(call.path) == "invoice.pdf" || filepath.Ext(call.path) != ".pdf" {
		t.Fatalf("upload path %q should be unique and keep .pdf", call.path)
	}
	if filepath.Dir(call.path) != filepath.Clean(uploadDir) {
		t.Fatalf("upload path %q not in upload dir %q", call.path, uploadDir)
	}
	if string(call.content) != "first" {
		t.Fatalf("content = %q", call.content)
	}
	assertUploadDirEmpty(t)
}

func TestSendMediaFileHeadlessConcurrentSameName(t *testing.T) {
	stub := newStub()
	// Hold both sends until both requests have returned, so both files
	// exist at the same time.
	stub.release = make(chan struct{})
	app := newTestApp(t, stub)

	contents := []string{"alpha", "bravo"}
	var wg sync.WaitGroup
	for _, content := range contents {
		wg.Add(1)
		go func(content string) {
			defer wg.Done()
			req := multipartRequest(t, map[string]string{"to": "919876543210", "msg": content}, "same.jpg", []byte(content), true)
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
			if err != nil {
				t.Errorf("app.Test: %v", err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("status %d", resp.StatusCode)
			}
		}(content)
	}
	wg.Wait()
	close(stub.release)

	paths := map[string]bool{}
	for range contents {
		call := waitCall(t, stub)
		if string(call.content) != call.msg {
			t.Fatalf("send for %q read %q: wrong file", call.msg, call.content)
		}
		if call.fileName != "same.jpg" {
			t.Fatalf("fileName = %q", call.fileName)
		}
		assertTimeout(t, call, perRecipientTimeout)
		paths[call.path] = true
		waitRemoved(t, call.path)
	}
	if len(paths) != 2 {
		t.Fatalf("uploads shared a path: %v", paths)
	}
	assertUploadDirEmpty(t)
}

// --- nil results from core (item 4) ---

func TestNilSendResultMapsToError(t *testing.T) {
	cases := []struct {
		name   string
		req    func(t *testing.T) *http.Request
		status int
		code   int
	}{
		{"base64", func(t *testing.T) *http.Request {
			return jsonRequest("/send_media_64", `{"to": ["919876543210"], "fileName": "a.pdf", "base64": "!!notbase64"}`, false)
		}, http.StatusBadRequest, interfaces.ERROR_INVALID_INPUT},
		{"url", func(t *testing.T) *http.Request {
			return jsonRequest("/send_media_url", `{"to": ["919876543210"], "url": "https://example.com/missing.pdf"}`, false)
		}, http.StatusBadGateway, interfaces.ERROR_INTERNAL_SERVER},
		{"upload", func(t *testing.T) *http.Request {
			return multipartRequest(t, map[string]string{"to": "919876543210"}, "a.pdf", []byte("x"), false)
		}, http.StatusInternalServerError, interfaces.ERROR_INTERNAL_SERVER},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := newStub()
			stub.nilResult = true
			app := newTestApp(t, stub)
			resp := do(t, app, tc.req(t))
			if resp.status != tc.status || resp.errorCode(t) != tc.code {
				t.Fatalf("got %d %s, want %d code %d", resp.status, resp.body, tc.status, tc.code)
			}
			waitCall(t, stub)
			assertUploadDirEmpty(t)
		})
	}
}

func TestSuccessfulMediaSendReturnsResult(t *testing.T) {
	stub := newStub()
	app := newTestApp(t, stub)
	resp := do(t, app, jsonRequest("/send_media_64", `{"to": ["919876543210"], "fileName": "a.pdf", "base64": "aGVsbG8="}`, false))
	if resp.status != http.StatusOK || !strings.Contains(string(resp.body), `"919876543210":true`) {
		t.Fatalf("got %d %s", resp.status, resp.body)
	}
	waitCall(t, stub)
}

// --- timeouts (item 5) ---

func TestSendTimeoutScalesAndCaps(t *testing.T) {
	cases := map[int]time.Duration{
		0:       perRecipientTimeout,
		1:       perRecipientTimeout,
		4:       4 * perRecipientTimeout,
		1000:    maxSendTimeout,
		1 << 40: maxSendTimeout,
	}
	for units, want := range cases {
		if got := sendTimeout(units); got != want {
			t.Errorf("sendTimeout(%d) = %v, want %v", units, got, want)
		}
	}
}

func TestSendTimeoutsPerHandler(t *testing.T) {
	three := `["919876543210","919876543211","919876543212"]`
	cases := []struct {
		name     string
		path     string
		body     string
		headless bool
		want     time.Duration
	}{
		{"text", "/send_message", `{"to": ` + three + `, "msg": "hi"}`, false, 3 * perRecipientTimeout},
		{"text headless", "/send_message", `{"to": ` + three + `, "msg": "hi"}`, true, 3 * perRecipientTimeout},
		{"base64", "/send_media_64", `{"to": ` + three + `, "fileName": "a.pdf", "base64": "aGVsbG8="}`, false, 3 * perRecipientTimeout},
		{"base64 headless", "/send_media_64", `{"to": ` + three + `, "fileName": "a.pdf", "base64": "aGVsbG8="}`, true, 3 * perRecipientTimeout},
		{"url", "/send_media_url", `{"to": ` + three + `, "url": "https://example.com/a.pdf"}`, false, 3 * perRecipientTimeout},
		{"url headless", "/send_media_url", `{"to": ` + three + `, "url": "https://example.com/a.pdf"}`, true, 3 * perRecipientTimeout},
		{"urls", "/send_media_url_multiple", `{"to": ` + three + `, "urls": ["https://example.com/a.pdf","https://example.com/b.pdf"]}`, false, 6 * perRecipientTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := newStub()
			app := newTestApp(t, stub)
			resp := do(t, app, jsonRequest(tc.path, tc.body, tc.headless))
			if resp.status != http.StatusOK {
				t.Fatalf("got %d %s, want 200", resp.status, resp.body)
			}
			assertTimeout(t, waitCall(t, stub), tc.want)
		})
	}
}

func TestSendMediaFileTimeoutScales(t *testing.T) {
	stub := newStub()
	app := newTestApp(t, stub)
	req := multipartRequest(t, map[string]string{"to": "919876543210,919876543211"}, "a.pdf", []byte("x"), false)
	if resp := do(t, app, req); resp.status != http.StatusOK {
		t.Fatalf("got %d %s", resp.status, resp.body)
	}
	assertTimeout(t, waitCall(t, stub), 2*perRecipientTimeout)
}
