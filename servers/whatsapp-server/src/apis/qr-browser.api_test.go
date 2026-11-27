package whatsapp_server_apis

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// scanPage requests /scan/<rawID> with rawID written to the request line as
// is, so characters such as <, " and ' reach the handler unescaped.
func scanPage(t *testing.T, rawID string) string {
	t.Helper()
	app := fiber.New()
	app.Get("/scan/:id", OpenBrowserWithQr)

	req := httptest.NewRequest("GET", "/scan/x", nil)
	req.URL.Opaque = "/scan/" + rawID
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

func TestOpenBrowserWithQrEscapesID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want string // the JS line the page must contain
	}{
		{"plain", "token-123", `const id = "token-123";`},
		{"script tag", "<script>alert(1)<", `const id = "\u003cscript\u003ealert(1)\u003c";`},
		{"double quote", `a";alert(1);"`, `const id = "a\";alert(1);\"";`},
		{"single quote", `a';alert(1);'`, `const id = "a';alert(1);'";`},
		{"backslash", `a\";alert(1)`, `const id = "a\\\";alert(1)";`},
		{"ampersand", "a&b", `const id = "a\u0026b";`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := scanPage(t, tc.id)
			if !strings.Contains(body, tc.want) {
				t.Fatalf("page does not contain %s\npage:\n%s", tc.want, body)
			}
			// The only <script> tags are the page's own.
			if n := strings.Count(body, "<script"); n != 1 {
				t.Fatalf("page has %d <script tags, want 1", n)
			}
			if n := strings.Count(body, "</script>"); n != 1 {
				t.Fatalf("page has %d </script> tags, want 1", n)
			}
		})
	}
}

// A path segment cannot hold "/", so "</script>" cannot arrive through the
// route. The percent-encoded form arrives as is and must stay inert.
func TestOpenBrowserWithQrEncodedScriptClose(t *testing.T) {
	body := scanPage(t, "a%3C%2Fscript%3E%22")
	want := `const id = "a%3C%2Fscript%3E%22";`
	if !strings.Contains(body, want) {
		t.Fatalf("page does not contain %s", want)
	}
	if n := strings.Count(body, "</script>"); n != 1 {
		t.Fatalf("page has %d </script> tags, want 1", n)
	}
}

// The escaping comes from json.Marshal; check that it covers a value with a
// closing script tag, which the route cannot deliver.
func TestQrPageIDEncodingCoversScriptClose(t *testing.T) {
	out, err := json.Marshal(`</script><script>alert(1)</script>`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(string(out), "<>") {
		t.Fatalf("json.Marshal left < or > in %s", out)
	}
}
