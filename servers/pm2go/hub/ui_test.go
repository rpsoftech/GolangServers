package hub

import (
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

var scriptSrc = regexp.MustCompile(`<script[^>]+src="([^"]+)"`)

// The dashboard is split into one file per page. Every script that index.html references must be
// served with a JavaScript content type, and no embedded script may be left unreferenced.
func TestUIServesIndexAndEveryScript(t *testing.T) {
	srv, _, _ := newAPI(t)
	get := func(path string) (string, string) {
		r, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			t.Fatalf("GET %s status = %d", path, r.StatusCode)
		}
		b, _ := io.ReadAll(r.Body)
		return string(b), r.Header.Get("Content-Type")
	}
	index, ct := get("/")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("/ content type = %q", ct)
	}
	refs := map[string]bool{}
	for _, m := range scriptSrc.FindAllStringSubmatch(index, -1) {
		refs[m[1]] = true
		body, ct := get("/" + m[1])
		if !strings.Contains(ct, "javascript") {
			t.Fatalf("%s content type = %q", m[1], ct)
		}
		if strings.TrimSpace(body) == "" {
			t.Fatalf("%s is empty", m[1])
		}
	}
	if len(refs) == 0 {
		t.Fatal("index.html references no scripts")
	}
	if _, ct := get("/style.css"); !strings.HasPrefix(ct, "text/css") {
		t.Fatalf("style.css content type = %q", ct)
	}
	files, _ := fs.Glob(uiFS, "ui/*.js")
	for _, f := range files {
		if name := strings.TrimPrefix(f, "ui/"); !refs[name] {
			t.Errorf("%s is embedded but not referenced by index.html", name)
		}
	}
}
