package hub

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeCert(t *testing.T, dir, cn string) (certFile, keyFile string) {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kb, _ := x509.MarshalECPrivateKey(key)
	certFile, keyFile = filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600)
	os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}), 0o600)
	return
}

func TestTLSOptionsValidate(t *testing.T) {
	bad := []TLSOptions{
		{CertFile: "c"},
		{KeyFile: "k"},
		{CertFile: "c", KeyFile: "k", ACMEDomain: "x.example.com"},
		{ACMEHTTP: ":80"},
	}
	for _, o := range bad {
		if o.Validate() == nil {
			t.Errorf("%+v must be rejected", o)
		}
	}
	for _, o := range []TLSOptions{{}, {CertFile: "c", KeyFile: "k"}, {ACMEDomain: "x.example.com", ACMEHTTP: ":8080"}} {
		if err := o.Validate(); err != nil {
			t.Errorf("%+v: %v", o, err)
		}
	}
}

func TestWarningsAndMode(t *testing.T) {
	tl := TLSOptions{CertFile: "c", KeyFile: "k"}
	if w := Warnings(tl, true, ":8443"); len(w) != 1 || !strings.Contains(w[0], "trust-proxy") {
		t.Fatalf("tls+proxy warnings = %v", w)
	}
	if w := Warnings(TLSOptions{}, false, ":8080"); len(w) != 1 || !strings.Contains(w[0], "clear text") {
		t.Fatalf("public plain http warnings = %v", w)
	}
	for _, quiet := range []struct {
		o      TLSOptions
		proxy  bool
		listen string
	}{{TLSOptions{}, false, "127.0.0.1:8080"}, {TLSOptions{}, false, "localhost:8080"}, {TLSOptions{}, true, ":8080"}, {tl, false, ":443"}} {
		if w := Warnings(quiet.o, quiet.proxy, quiet.listen); len(w) != 0 {
			t.Errorf("%+v: unexpected warnings %v", quiet, w)
		}
	}
	if Mode(TLSOptions{}, false) != "plain HTTP" || !strings.Contains(Mode(tl, false), "TLS") || !strings.Contains(Mode(TLSOptions{}, true), "proxy") {
		t.Fatal("mode strings")
	}
}

func TestHubServesHTTPSAndReloadsCertificate(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeCert(t, dir, "first")
	rl, err := newCertReloader(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	rl.checkEvery = 0 // check the files on every handshake

	_, api, _ := newAPI(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: hsts(api.Handler()), TLSConfig: rl.tlsConfig()}
	go srv.ServeTLS(ln, "", "")
	defer srv.Close()
	addr, base := ln.Addr().String(), "https://"+ln.Addr().String()

	leafCN := func() (string, *http.Response) {
		conn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		cn := conn.ConnectionState().PeerCertificates[0].Subject.CommonName
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		r, err := client.Post(base+"/api/login", "application/json", strings.NewReader(`{"username":"admin","password":"pw"}`))
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		return cn, r
	}
	cn, r := leafCN()
	if cn != "first" {
		t.Fatalf("cn = %q", cn)
	}
	if r.Header.Get("Strict-Transport-Security") == "" {
		t.Fatal("HSTS header missing")
	}
	var secure bool
	for _, c := range r.Cookies() {
		if c.Name == cookieName {
			secure = c.Secure
		}
	}
	if !secure {
		t.Fatalf("session cookie must be Secure over TLS (status %d)", r.StatusCode)
	}

	// Renew: overwrite the files with a new certificate, no restart.
	writeCert(t, dir, "second")
	future := time.Now().Add(2 * time.Second)
	os.Chtimes(certFile, future, future)
	os.Chtimes(keyFile, future, future)
	if cn, _ := leafCN(); cn != "second" {
		t.Fatalf("after renewal cn = %q", cn)
	}

	// A broken renewal keeps the old certificate.
	os.WriteFile(certFile, []byte("garbage"), 0o600)
	later := future.Add(2 * time.Second)
	os.Chtimes(certFile, later, later)
	if cn, _ := leafCN(); cn != "second" {
		t.Fatalf("after broken renewal cn = %q", cn)
	}
}

func TestRedirectToHTTPS(t *testing.T) {
	h := redirectToHTTPS(":8443")
	req := httptest.NewRequest("GET", "http://hub.example.com:8080/a/b?x=1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusPermanentRedirect || w.Header().Get("Location") != "https://hub.example.com:8443/a/b?x=1" {
		t.Fatalf("%d %s", w.Code, w.Header().Get("Location"))
	}
	w = httptest.NewRecorder()
	redirectToHTTPS(":443").ServeHTTP(w, httptest.NewRequest("GET", "http://hub.example.com/", nil))
	if w.Header().Get("Location") != "https://hub.example.com/" {
		t.Fatal(w.Header().Get("Location"))
	}
}

func TestACMESetupBuildsConfig(t *testing.T) {
	s, err := newTLSSetup(TLSOptions{ACMEDomain: "hub.example.com"}, t.TempDir(), ":443")
	if err != nil {
		t.Fatal(err)
	}
	if s.config == nil || s.config.GetCertificate == nil || s.httpSrv == nil || s.httpSrv.Addr != ":80" {
		t.Fatalf("acme setup incomplete: %+v", s)
	}
	// The HTTP listener redirects ordinary requests to HTTPS.
	w := httptest.NewRecorder()
	s.httpSrv.Handler.ServeHTTP(w, httptest.NewRequest("GET", "http://hub.example.com/x", nil))
	if w.Code != http.StatusPermanentRedirect {
		t.Fatalf("code %d", w.Code)
	}
}
