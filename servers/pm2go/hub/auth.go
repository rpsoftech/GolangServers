package hub

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cookieName = "pm2go_session"
	sessionTTL = 7 * 24 * time.Hour
)

type auth struct {
	cfg Config

	mu    sync.Mutex
	fails map[string]failure
}

type failure struct {
	n    int
	last time.Time
}

func newAuth(cfg Config) *auth { return &auth{cfg: cfg, fails: map[string]failure{}} }

func eq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

func (a *auth) sign(exp int64) string {
	m := hmac.New(sha256.New, []byte(a.cfg.SessionSecret))
	fmt.Fprint(m, exp)
	return strconv.FormatInt(exp, 10) + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (a *auth) validToken(tok string) bool {
	i := strings.IndexByte(tok, '.')
	if i < 0 {
		return false
	}
	exp, err := strconv.ParseInt(tok[:i], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return eq(tok, a.sign(exp))
}

// authed accepts a session cookie or "Authorization: Bearer <admin password>".
func (a *auth) authed(r *http.Request) bool {
	if c, err := r.Cookie(cookieName); err == nil && a.validToken(c.Value) {
		return true
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return eq(strings.TrimPrefix(h, "Bearer "), a.cfg.AdminPassword)
	}
	return false
}

func clientIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

// login checks the password with a per-IP lockout that grows with failures.
func (a *auth) login(w http.ResponseWriter, r *http.Request, password string) bool {
	ip := clientIP(r)
	a.mu.Lock()
	f := a.fails[ip]
	wait := time.Duration(min(f.n, 6)) * 2 * time.Second
	if f.n > 0 && time.Since(f.last) < wait {
		a.mu.Unlock()
		http.Error(w, "too many attempts, wait a moment", http.StatusTooManyRequests)
		return false
	}
	a.mu.Unlock()
	if !eq(password, a.cfg.AdminPassword) {
		a.mu.Lock()
		f.n++
		f.last = time.Now()
		a.fails[ip] = f
		a.mu.Unlock()
		http.Error(w, "wrong password", http.StatusUnauthorized)
		return false
	}
	a.mu.Lock()
	delete(a.fails, ip)
	a.mu.Unlock()
	exp := time.Now().Add(sessionTTL)
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: a.sign(exp.Unix()), Path: "/", Expires: exp,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil})
	return true
}

func (a *auth) logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
}

func (a *auth) require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authed(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *auth) agentKeyOK(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	return strings.HasPrefix(h, "Bearer ") && eq(strings.TrimPrefix(h, "Bearer "), a.cfg.AgentKey)
}
