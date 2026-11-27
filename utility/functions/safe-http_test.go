package utility_functions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// allowOnly lets FetchUserURL dial exactly the given test servers. Every
// other address still goes through the production check.
func allowOnly(t *testing.T, servers ...*httptest.Server) {
	t.Helper()
	allowed := map[string]bool{}
	for _, s := range servers {
		allowed[s.Listener.Addr().String()] = true
	}
	allowDialForTest = func(address string) bool { return allowed[address] }
	t.Cleanup(func() {
		allowDialForTest = nil
		userURLClient.CloseIdleConnections()
	})
}

func setMaxBytes(t *testing.T, n int64) {
	t.Helper()
	maxUserURLBytes = n
	t.Cleanup(func() { maxUserURLBytes = MaxUserURLDownloadBytes })
}

func fetch(t *testing.T, rawURL string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return FetchUserURL(ctx, rawURL)
}

func requireBlocked(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("err = %v, want ErrBlockedAddress", err)
	}
}

func okServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, body)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestProductionDefaultsAreStrict(t *testing.T) {
	if allowDialForTest != nil {
		t.Fatal("allowDialForTest must be nil by default")
	}
	if maxUserURLBytes != MaxUserURLDownloadBytes {
		t.Fatalf("maxUserURLBytes = %d, want %d", maxUserURLBytes, MaxUserURLDownloadBytes)
	}
}

func TestBlockNonPublicAddress(t *testing.T) {
	blocked := []string{
		"127.0.0.1:80", "127.1.2.3:80", "[::1]:80",
		"10.0.0.1:80", "172.16.0.1:80", "192.168.1.1:80", "[fc00::1]:80",
		"169.254.169.254:80", "169.254.1.1:80", "[fe80::1]:80",
		"100.64.0.1:80", "0.0.0.0:80", "[::]:80",
		"224.0.0.1:80", "255.255.255.255:80",
		"[::ffff:127.0.0.1]:80", "[::ffff:169.254.169.254]:80", "[::ffff:10.0.0.1]:80",
	}
	for _, address := range blocked {
		if err := blockNonPublicAddress("tcp", address, nil); !errors.Is(err, ErrBlockedAddress) {
			t.Errorf("%s: err = %v, want ErrBlockedAddress", address, err)
		}
	}
	allowed := []string{"8.8.8.8:443", "1.1.1.1:80", "[2606:4700:4700::1111]:443"}
	for _, address := range allowed {
		if err := blockNonPublicAddress("tcp", address, nil); err != nil {
			t.Errorf("%s: err = %v, want nil", address, err)
		}
	}
}

func TestFetchUserURLBlocksLoopbackServer(t *testing.T) {
	// A live server: if the check failed, the fetch would succeed.
	s := okServer(t, "secret")
	_, err := fetch(t, s.URL)
	requireBlocked(t, err)
}

func TestFetchUserURLBlocksNonPublicLiterals(t *testing.T) {
	for _, rawURL := range []string{
		"http://127.0.0.1/",
		"http://[::1]/",
		"http://10.1.2.3/",
		"http://192.168.0.10:8080/",
		"http://172.20.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::ffff:169.254.169.254]/",
		"http://100.100.100.200/",
		"http://0.0.0.0/",
	} {
		t.Run(rawURL, func(t *testing.T) {
			_, err := fetch(t, rawURL)
			requireBlocked(t, err)
		})
	}
}

func TestFetchUserURLBlocksNameResolvingToLoopback(t *testing.T) {
	s := okServer(t, "secret")
	_, port, _ := net.SplitHostPort(s.Listener.Addr().String())
	_, err := fetch(t, "http://localhost:"+port+"/")
	requireBlocked(t, err)
}

func TestFetchUserURLRejectsSchemes(t *testing.T) {
	for _, rawURL := range []string{"file:///etc/passwd", "ftp://example.com/x", "gopher://example.com/", "/relative", "://bad"} {
		if _, err := fetch(t, rawURL); err == nil {
			t.Errorf("%s: want error", rawURL)
		}
	}
}

func TestFetchUserURLAllowedServer(t *testing.T) {
	s := okServer(t, "hello")
	allowOnly(t, s)
	data, err := fetch(t, s.URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("body = %q, want hello", data)
	}
}

func TestFetchUserURLNon200(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(s.Close)
	allowOnly(t, s)
	if _, err := fetch(t, s.URL); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err = %v, want status 404 error", err)
	}
}

func redirectServer(t *testing.T, location string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, location, http.StatusFound)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestFetchUserURLBlocksRedirectToLoopback(t *testing.T) {
	internal := okServer(t, "secret")
	public := redirectServer(t, internal.URL+"/admin")
	allowOnly(t, public) // the redirect target is not allowed
	data, err := fetch(t, public.URL)
	requireBlocked(t, err)
	if len(data) != 0 {
		t.Fatalf("got data %q from blocked redirect", data)
	}
}

func TestFetchUserURLBlocksRedirectToMetadata(t *testing.T) {
	public := redirectServer(t, "http://169.254.169.254/latest/meta-data/")
	allowOnly(t, public)
	_, err := fetch(t, public.URL)
	requireBlocked(t, err)
}

func TestFetchUserURLBlocksRedirectToNameResolvingToLoopback(t *testing.T) {
	internal := okServer(t, "secret")
	_, port, _ := net.SplitHostPort(internal.Listener.Addr().String())
	public := redirectServer(t, "http://localhost:"+port+"/")
	allowOnly(t, public)
	_, err := fetch(t, public.URL)
	requireBlocked(t, err)
}

func TestFetchUserURLBlocksRedirectToOtherScheme(t *testing.T) {
	public := redirectServer(t, "file:///etc/passwd")
	allowOnly(t, public)
	_, err := fetch(t, public.URL)
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("err = %v, want scheme error", err)
	}
}

func TestFetchUserURLFollowsAllowedRedirect(t *testing.T) {
	target := okServer(t, "moved")
	public := redirectServer(t, target.URL)
	allowOnly(t, public, target)
	data, err := fetch(t, public.URL)
	if err != nil || string(data) != "moved" {
		t.Fatalf("data, err = %q, %v; want moved, nil", data, err)
	}
}

func TestFetchUserURLStopsRedirectLoop(t *testing.T) {
	var s *httptest.Server
	s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.URL+r.URL.Path+"x", http.StatusFound)
	}))
	t.Cleanup(s.Close)
	allowOnly(t, s)
	if _, err := fetch(t, s.URL+"/"); err == nil || !strings.Contains(err.Error(), "redirects") {
		t.Fatalf("err = %v, want redirect limit error", err)
	}
}

func TestFetchUserURLSizeCap(t *testing.T) {
	const limit = 1024
	setMaxBytes(t, limit)

	fixed := func(size int) *httptest.Server {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprint(size))
			w.Write(bytes.Repeat([]byte("a"), size))
		}))
		t.Cleanup(s.Close)
		return s
	}
	// No Content-Length: the body arrives chunked and only the read limit
	// can stop it.
	chunked := func(size int) *httptest.Server {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.(http.Flusher).Flush()
			for i := 0; i < size; i += 100 {
				n := min(100, size-i)
				w.Write(bytes.Repeat([]byte("b"), n))
				w.(http.Flusher).Flush()
			}
		}))
		t.Cleanup(s.Close)
		return s
	}

	cases := []struct {
		name   string
		server *httptest.Server
		size   int
		tooBig bool
	}{
		{"content-length at limit", fixed(limit), limit, false},
		{"content-length over limit", fixed(limit + 1), limit + 1, true},
		{"chunked at limit", chunked(limit), limit, false},
		{"chunked over limit", chunked(limit + 1), limit + 1, true},
		{"chunked far over limit", chunked(limit * 50), limit * 50, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			allowOnly(t, tc.server)
			data, err := fetch(t, tc.server.URL)
			if tc.tooBig {
				if !errors.Is(err, ErrDownloadTooBig) {
					t.Fatalf("err = %v, want ErrDownloadTooBig", err)
				}
				return
			}
			if err != nil || len(data) != tc.size {
				t.Fatalf("len(data), err = %d, %v; want %d, nil", len(data), err, tc.size)
			}
		})
	}
}
