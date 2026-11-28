package hub

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// TLSOptions selects the built-in TLS mode. The zero value means plain HTTP.
type TLSOptions struct {
	CertFile   string // PEM certificate (with chain)
	KeyFile    string // PEM private key
	ACMEDomain string // obtain certificates with Let's Encrypt for this host name
	ACMEHTTP   string // plain HTTP listener for HTTP-01 and redirects, e.g. ":80"
}

// Enabled reports whether the hub terminates TLS itself.
func (o TLSOptions) Enabled() bool { return o.CertFile != "" || o.ACMEDomain != "" }

// Validate rejects inconsistent flag combinations.
func (o TLSOptions) Validate() error {
	if (o.CertFile == "") != (o.KeyFile == "") {
		return errors.New("--tls-cert and --tls-key must be given together")
	}
	if o.CertFile != "" && o.ACMEDomain != "" {
		return errors.New("use either --tls-cert/--tls-key or --acme-domain, not both")
	}
	if o.ACMEHTTP != "" && !o.Enabled() {
		return errors.New("--acme-http needs --tls-cert/--tls-key or --acme-domain")
	}
	return nil
}

// Mode names the serving mode for the startup log line.
func Mode(o TLSOptions, proxy bool) string {
	switch {
	case o.ACMEDomain != "":
		return "TLS (automatic certificates for " + o.ACMEDomain + ")"
	case o.CertFile != "":
		return "TLS (certificate files)"
	case proxy:
		return "plain HTTP behind a trusted proxy"
	}
	return "plain HTTP"
}

// Warnings lists risky combinations to print at startup.
func Warnings(o TLSOptions, proxy bool, listen string) []string {
	var w []string
	if o.Enabled() && proxy {
		w = append(w, "trust-proxy is on together with built-in TLS: clients that reach the hub directly can forge X-Forwarded-For; use only one of them")
	}
	if !o.Enabled() && !proxy && !loopbackAddr(listen) {
		w = append(w, fmt.Sprintf("listening on %s in plain HTTP without --trust-proxy: the admin password, session cookie and agent key travel in clear text; enable --tls-cert/--tls-key or --acme-domain, or put a TLS proxy in front", listen))
	}
	return w
}

// loopbackAddr reports whether a listen address only accepts local clients.
func loopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// certReloader serves a certificate from PEM files and re-reads them when the
// files change (checked at most every checkEvery on a handshake) or when
// Reload is called, for example on SIGHUP.
type certReloader struct {
	certFile, keyFile string
	checkEvery        time.Duration

	mu      sync.Mutex
	cert    *tls.Certificate
	certMod time.Time
	keyMod  time.Time
	checked time.Time
}

func newCertReloader(certFile, keyFile string) (*certReloader, error) {
	c := &certReloader{certFile: certFile, keyFile: keyFile, checkEvery: 10 * time.Second}
	if err := c.Reload(); err != nil {
		return nil, err
	}
	return c, nil
}

func modTime(path string) time.Time {
	if st, err := os.Stat(path); err == nil {
		return st.ModTime()
	}
	return time.Time{}
}

// Reload loads the files now. On error the previous certificate stays in use.
func (c *certReloader) Reload() error {
	cm, km := modTime(c.certFile), modTime(c.keyFile)
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.cert, c.certMod, c.keyMod, c.checked = &cert, cm, km, time.Now()
	c.mu.Unlock()
	return nil
}

// GetCertificate is the tls.Config callback.
func (c *certReloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.mu.Lock()
	due := time.Since(c.checked) >= c.checkEvery
	if due {
		c.checked = time.Now()
	}
	cm, km := c.certMod, c.keyMod
	c.mu.Unlock()
	if due && (!modTime(c.certFile).Equal(cm) || !modTime(c.keyFile).Equal(km)) {
		if err := c.Reload(); err != nil {
			log.Printf("pm2go hub: certificate reload failed, keeping the old one: %v", err)
		} else {
			log.Print("pm2go hub: TLS certificate reloaded")
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cert, nil
}

func (c *certReloader) tlsConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: c.GetCertificate}
}

// tlsSetup holds what Run needs for the chosen TLS mode.
type tlsSetup struct {
	config  *tls.Config
	reload  func() error // nil in ACME mode
	httpSrv *http.Server // optional plain HTTP listener (redirect, HTTP-01)
}

func newTLSSetup(o TLSOptions, home, listen string) (*tlsSetup, error) {
	s := &tlsSetup{}
	var challenge func(http.Handler) http.Handler
	if o.ACMEDomain != "" {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(o.ACMEDomain),
			Cache:      autocert.DirCache(filepath.Join(home, "acme")),
		}
		s.config = m.TLSConfig()
		s.config.MinVersion = tls.VersionTLS12
		challenge = m.HTTPHandler
		if o.ACMEHTTP == "" {
			o.ACMEHTTP = ":80"
		}
	} else {
		rl, err := newCertReloader(o.CertFile, o.KeyFile)
		if err != nil {
			return nil, err
		}
		s.config, s.reload = rl.tlsConfig(), rl.Reload
	}
	if o.ACMEHTTP != "" {
		var h http.Handler = redirectToHTTPS(listen)
		if challenge != nil {
			h = challenge(h)
		}
		s.httpSrv = &http.Server{Addr: o.ACMEHTTP, Handler: h, ReadHeaderTimeout: 10 * time.Second}
	}
	return s, nil
}

// redirectToHTTPS answers every request with a redirect to the same URL over
// HTTPS on the hub's TLS port.
func redirectToHTTPS(tlsListen string) http.Handler {
	_, port, _ := net.SplitHostPort(tlsListen)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
			host = "[" + host + "]" // bare IPv6 literal
		}
		if port != "" && port != "443" {
			host += ":" + port
		}
		http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusPermanentRedirect)
	})
}

// hsts adds Strict-Transport-Security to every response.
func hsts(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		next.ServeHTTP(w, r)
	})
}
