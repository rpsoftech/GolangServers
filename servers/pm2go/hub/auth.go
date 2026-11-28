package hub

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	cookieName = "pm2go_session"
	sessionTTL = 7 * 24 * time.Hour
)

type auth struct {
	cfg   Config
	store *Store

	mu    sync.Mutex
	fails map[string]failure
}

type failure struct {
	n    int
	last time.Time
}

// newAuth creates the auth layer and, when no users exist yet, the bootstrap
// admin from config.json.
func newAuth(cfg Config, store *Store) (*auth, error) {
	if err := store.BootstrapAdmin(cfg.AdminPassword); err != nil {
		return nil, err
	}
	return &auth{cfg: cfg, store: store, fails: map[string]failure{}}, nil
}

func eq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

// principal is who a request acts as. Role is the effective role: the user's
// role, lowered by the token's role_cap for token requests.
type principal struct {
	User    User
	Role    string
	Via     string // "session" or "token"
	session string // session id hash (session requests)
	tokenID int64
}

type ctxKey struct{}

func principalOf(r *http.Request) *principal {
	p, _ := r.Context().Value(ctxKey{}).(*principal)
	return p
}

// authenticate resolves "Authorization: Bearer pm2go_..." or the session cookie.
func (a *auth) authenticate(r *http.Request) *principal {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		if u, t, ok := a.store.LookupToken(strings.TrimPrefix(h, "Bearer ")); ok {
			return &principal{User: u, Role: minRole(u.Role, t.RoleCap), Via: "token", tokenID: t.ID}
		}
		return nil
	}
	if c, err := r.Cookie(cookieName); err == nil {
		if u, id, ok := a.store.LookupSession(c.Value); ok {
			return &principal{User: u, Role: u.Role, Via: "session", session: id}
		}
	}
	return nil
}

// guard is the single permission check. Every authenticated route goes through
// it with the minimum role it needs (the route table is in API.Handler).
// allowMustChange lets a user who must still change the password reach the
// route; sessionOnly rejects API tokens (used for credential management).
func (a *auth) guard(min string, allowMustChange, sessionOnly bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := a.authenticate(r)
		if p == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if p.User.MustChange && !allowMustChange {
			http.Error(w, "password change required", http.StatusForbidden)
			return
		}
		if sessionOnly && p.Via != "session" {
			http.Error(w, "forbidden: sign in with a session, not an API token", http.StatusForbidden)
			return
		}
		if roleRank[p.Role] < roleRank[min] {
			http.Error(w, "forbidden: requires role "+min, http.StatusForbidden)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, p)))
	}
}

func (a *auth) requireRole(min string, next http.HandlerFunc) http.HandlerFunc {
	return a.guard(min, false, false, next)
}

// requireSession is requireRole for routes that manage credentials: API tokens
// are refused so a stolen token cannot mint more tokens or change passwords.
func (a *auth) requireSession(min string, next http.HandlerFunc) http.HandlerFunc {
	return a.guard(min, false, true, next)
}

func (a *auth) agentKeyOK(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	return strings.HasPrefix(h, "Bearer ") && eq(strings.TrimPrefix(h, "Bearer "), a.cfg.AgentKey)
}

// trustProxy makes the hub believe X-Forwarded-For and X-Forwarded-Proto. Turn
// it on only when every request comes through a reverse proxy you control
// (Caddy), otherwise clients can forge their address.
var trustProxy bool

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || (trustProxy && r.Header.Get("X-Forwarded-Proto") == "https")
}

func clientIP(r *http.Request) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first, _, _ := strings.Cut(xff, ",")
			if ip := strings.TrimSpace(first); net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

// ---- login with backoff ----

const failIdle = 10 * time.Minute

// wait returns how long key must still wait: 2 s per recent failure, up to 12 s.
func (a *auth) wait(key string) time.Duration {
	f, ok := a.fails[key]
	if !ok {
		return 0
	}
	if time.Since(f.last) > failIdle {
		delete(a.fails, key)
		return 0
	}
	if d := time.Duration(min(f.n, 6))*2*time.Second - time.Since(f.last); d > 0 {
		return d
	}
	return 0
}

func (a *auth) fail(keys ...string) {
	if len(a.fails) > 5000 { // bound memory under a spray of random usernames
		for k, f := range a.fails {
			if time.Since(f.last) > failIdle {
				delete(a.fails, k)
			}
		}
	}
	for _, k := range keys {
		f := a.fails[k]
		f.n++
		f.last = time.Now()
		a.fails[k] = f
	}
}

// login verifies the password and starts a session. Backoff is per IP and
// username (a targeted guess) and per IP alone (guessing across usernames).
// It writes the error response itself and audits every outcome.
func (a *auth) login(w http.ResponseWriter, r *http.Request, username, password string) (User, bool) {
	ip := clientIP(r)
	userKey := ip + "|" + strings.ToLower(username)
	a.mu.Lock()
	if a.wait(userKey) > 0 || a.wait(ip) > 0 {
		a.mu.Unlock()
		http.Error(w, "too many attempts, wait a moment", http.StatusTooManyRequests)
		return User{}, false
	}
	a.mu.Unlock()

	u, hash, err := a.store.userWithHash(username)
	hashToCheck := hash
	if err != nil {
		hashToCheck = dummyHash // same cost whether or not the user exists
	}
	ok := verifyPassword(hashToCheck, password) && err == nil && !u.Disabled
	if !ok {
		a.mu.Lock()
		a.fail(userKey, ip)
		a.mu.Unlock()
		target, detail := "unknown", "unknown user or wrong password"
		if err == nil { // log the name only when it is a real account
			target = u.Username
			detail = "wrong password"
			if u.Disabled {
				detail = "account disabled"
			}
		}
		a.store.Audit(AuditEntry{User: target, IP: ip, Action: "login", Target: target, Detail: detail, Success: false})
		http.Error(w, "wrong username or password", http.StatusUnauthorized)
		return User{}, false
	}
	a.mu.Lock()
	delete(a.fails, userKey)
	a.mu.Unlock()
	raw, exp, err := a.store.CreateSession(u.ID, ip, r.UserAgent())
	if err != nil {
		http.Error(w, "could not start session", http.StatusInternalServerError)
		return User{}, false
	}
	a.store.TouchLogin(u.ID)
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: raw, Path: "/", Expires: exp,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: isHTTPS(r)})
	a.store.Audit(AuditEntry{User: u.Username, IP: ip, Action: "login", Target: u.Username, Success: true})
	return u, true
}

// logout deletes the server-side session and clears the cookie.
func (a *auth) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		if u, id, ok := a.store.LookupSession(c.Value); ok {
			a.store.DeleteSession(id)
			a.store.Audit(AuditEntry{User: u.Username, IP: clientIP(r), Action: "logout", Target: u.Username, Success: true})
		}
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		SameSite: http.SameSiteStrictMode, Secure: isHTTPS(r)})
}
