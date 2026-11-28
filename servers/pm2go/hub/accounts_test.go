package hub

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func init() { argonMem, argonThreads = 1024, 1 } // fast hashing in tests; parameters are stored per hash

// ---- helpers ----

func mkUser(t *testing.T, a *API, name, role string) User {
	t.Helper()
	u, err := a.store.CreateUser(name, "password-"+name, role, false)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// mkToken returns an API token secret for an existing or new user with the given cap.
func mkToken(t *testing.T, a *API, user, roleCap string) string {
	t.Helper()
	u, _, err := a.store.userWithHash(user)
	if err != nil {
		t.Fatal(err)
	}
	_, secret, err := a.store.CreateToken(u.ID, "test", roleCap, 0)
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

// readyAdmin returns the bootstrap admin after its forced password change.
func readyAdmin(t *testing.T, a *API) User {
	t.Helper()
	u, _, err := a.store.userWithHash("admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.store.SetPassword(u.ID, "admin-changed-1", false, ""); err != nil {
		t.Fatal(err)
	}
	u.MustChange = false
	return u
}

func mkSession(t *testing.T, a *API, u User) string {
	t.Helper()
	raw, _, err := a.store.CreateSession(u.ID, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

type authz struct{ cookie, bearer string }

func do(t *testing.T, srv *httptest.Server, method, path, body string, z authz) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if z.cookie != "" {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: z.cookie})
	}
	if z.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+z.bearer)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Body.Close() })
	return r
}

func bodyOf(r *http.Response) string { b, _ := io.ReadAll(r.Body); return string(b) }

// ---- route to role matrix ----

type route struct {
	method, pattern, body string
	min                   string // viewer, operator, admin
	sessionOnly           bool   // API tokens are refused
}

var routeMatrix = []route{
	{"GET", "/api/servers", "", roleViewer, false},
	{"GET", "/api/servers/box1", "", roleViewer, false},
	{"GET", "/api/servers/box1/metrics", "", roleViewer, false},
	{"GET", "/api/servers/box1/logs", "", roleViewer, false},
	{"POST", "/api/servers/box1/command", `{"action":"restart","target":"x"}`, roleOperator, false},
	{"GET", "/api/logs/ws", "", roleViewer, false},
	{"GET", "/api/events", "", roleViewer, false},
	{"GET", "/api/issues", "", roleViewer, false},
	{"POST", "/api/issues/status", `{"status":"open"}`, roleOperator, false},
	{"GET", "/api/alerts/rules", "", roleViewer, false},
	{"POST", "/api/alerts/rules", `{}`, roleAdmin, false},
	{"PUT", "/api/alerts/rules/9999", `{}`, roleAdmin, false},
	{"DELETE", "/api/alerts/rules/9999", "", roleAdmin, false},
	{"GET", "/api/alerts/active", "", roleViewer, false},
	{"GET", "/api/alerts/history", "", roleViewer, false},
	{"GET", "/api/settings/info", "", roleViewer, false},
	{"GET", "/api/settings/telegram", "", roleAdmin, false},
	{"PUT", "/api/settings/telegram", `{"chats":[{"name":"x"}]}`, roleAdmin, false},
	{"POST", "/api/settings/telegram/test", `{}`, roleAdmin, false},
	{"GET", "/api/users", "", roleAdmin, false},
	{"POST", "/api/users", `{}`, roleAdmin, true},
	{"PUT", "/api/users/9999", `{}`, roleAdmin, true},
	{"DELETE", "/api/users/9999", "", roleAdmin, true},
	{"POST", "/api/users/9999/password", `{"password":"longenough1"}`, roleAdmin, true},
	{"POST", "/api/users/9999/revoke-sessions", "", roleAdmin, true},
	{"GET", "/api/tokens", "", roleViewer, true},
	{"POST", "/api/tokens", `{}`, roleViewer, true},
	{"DELETE", "/api/tokens/9999", "", roleViewer, true},
	{"POST", "/api/sessions/revoke-all", "", roleViewer, true},
	{"POST", "/api/password", `{}`, roleViewer, true},
	{"GET", "/api/me", "", roleViewer, false},
	{"GET", "/api/audit", "", roleAdmin, false},
}

// publicRoutes need no sign-in (the agent socket uses the agent key).
var publicRoutes = map[string]bool{"GET /": true, "GET /healthz": true, "GET /agent/ws": true, "POST /api/login": true, "POST /api/logout": true}

// Every route registered in api.go must appear in the matrix or the public list.
func TestRouteMatrixCoversEveryRoute(t *testing.T) {
	src, err := os.ReadFile("api.go")
	if err != nil {
		t.Fatal(err)
	}
	inMatrix := map[string]bool{}
	for _, r := range routeMatrix {
		p := r.pattern
		inMatrix[r.method+" "+regexp.MustCompile(`/\d+|/box1`).ReplaceAllStringFunc(p, func(s string) string {
			if s == "/box1" {
				return "/{name}"
			}
			return "/{id}"
		})] = true
	}
	for _, m := range regexp.MustCompile(`mux\.Handle(?:Func)?\("([A-Z]+ [^"]+)"`).FindAllStringSubmatch(string(src), -1) {
		if publicRoutes[m[1]] {
			continue
		}
		// metrics/logs/command routes keep {name} mid-path
		key := m[1]
		if !inMatrix[key] {
			t.Errorf("route %q is in api.go but not in routeMatrix: give it a role and a test", key)
		}
	}
}

func TestRoleMatrix(t *testing.T) {
	srv, api, _ := newAPI(t)
	users := map[string]User{}
	for _, role := range []string{roleViewer, roleOperator, roleAdmin} {
		users[role] = mkUser(t, api, "u-"+role, role)
	}
	sess := map[string]authz{}
	tok := map[string]authz{}
	for role, u := range users {
		sess[role] = authz{cookie: mkSession(t, api, u)}
		tok[role] = authz{bearer: mkToken(t, api, u.Username, role)}
	}
	for _, rt := range routeMatrix {
		name := rt.method + " " + rt.pattern
		if r := do(t, srv, rt.method, rt.pattern, rt.body, authz{}); r.StatusCode != 401 {
			t.Errorf("%s without credentials = %d, want 401", name, r.StatusCode)
		}
		if r := do(t, srv, rt.method, rt.pattern, rt.body, authz{bearer: "pm2go_nonsense-token"}); r.StatusCode != 401 {
			t.Errorf("%s with a bad token = %d, want 401", name, r.StatusCode)
		}
		for _, role := range []string{roleViewer, roleOperator, roleAdmin} {
			allowed := roleRank[role] >= roleRank[rt.min]
			check := func(via string, z authz, allowed bool) {
				code := do(t, srv, rt.method, rt.pattern, rt.body, z).StatusCode
				if allowed && (code == 401 || code == 403) {
					t.Errorf("%s as %s via %s = %d, want allowed", name, role, via, code)
				}
				if !allowed && code != 403 {
					t.Errorf("%s as %s via %s = %d, want 403", name, role, via, code)
				}
			}
			check("session", sess[role], allowed)
			check("token", tok[role], allowed && !rt.sessionOnly)
		}
	}
}

func TestOperatorCannotDeleteProcessButCanRestart(t *testing.T) {
	srv, api, _ := newAPI(t)
	op := authz{cookie: mkSession(t, api, mkUser(t, api, "op", roleOperator))}
	ad := authz{cookie: mkSession(t, api, mkUser(t, api, "ad", roleAdmin))}
	if r := do(t, srv, "POST", "/api/servers/box1/command", `{"action":"delete","target":"x"}`, op); r.StatusCode != 403 {
		t.Fatalf("operator delete = %d", r.StatusCode)
	}
	if r := do(t, srv, "POST", "/api/servers/box1/command", `{"action":"delete","target":"x"}`, ad); r.StatusCode != 502 { // passes the role check, server offline
		t.Fatalf("admin delete = %d", r.StatusCode)
	}
	if r := do(t, srv, "POST", "/api/servers/box1/command", `{"action":"restart","target":"x"}`, op); r.StatusCode != 502 {
		t.Fatalf("operator restart = %d", r.StatusCode)
	}
}

func TestViewerNeverSeesSecrets(t *testing.T) {
	srv, api, cfg := newAPI(t)
	if err := api.notif.Update(TelegramSettings{Enabled: true, BotToken: "SECRET-BOT-TOKEN", Chats: []TelegramChat{{Name: "c", ChatID: "1", MinLevel: "info"}}}); err != nil {
		t.Fatal(err)
	}
	v := authz{cookie: mkSession(t, api, mkUser(t, api, "v", roleViewer))}
	a := authz{cookie: mkSession(t, api, mkUser(t, api, "a", roleAdmin))}
	for _, p := range []string{"/api/settings/info", "/api/settings/telegram", "/api/alerts/rules", "/api/users", "/api/tokens", "/api/me"} {
		b := bodyOf(do(t, srv, "GET", p, "", v))
		if strings.Contains(b, cfg.AgentKey) || strings.Contains(b, "SECRET-BOT-TOKEN") || strings.Contains(b, "password_hash") || strings.Contains(b, "argon2") {
			t.Errorf("viewer sees a secret at %s: %s", p, b)
		}
	}
	if b := bodyOf(do(t, srv, "GET", "/api/settings/info", "", a)); !strings.Contains(b, cfg.AgentKey) {
		t.Errorf("admin should see the agent key: %s", b)
	}
	if b := bodyOf(do(t, srv, "GET", "/api/users", "", a)); strings.Contains(b, "argon2") || strings.Contains(b, "password_hash") {
		t.Errorf("user list leaks hashes: %s", b)
	}
}

// ---- bootstrap, upgrade, login ----

func login(t *testing.T, srv *httptest.Server, user, pw string) (*http.Response, string) {
	t.Helper()
	r := do(t, srv, "POST", "/api/login", `{"username":"`+user+`","password":"`+pw+`"}`, authz{})
	for _, c := range r.Cookies() {
		if c.Name == cookieName && c.Value != "" {
			return r, c.Value
		}
	}
	return r, ""
}

func TestBootstrapAdminFromConfigPasswordForcesChange(t *testing.T) {
	srv, api, _ := newAPI(t)
	r, cookie := login(t, srv, "admin", "pw") // admin_password from config.json
	if r.StatusCode != 200 || cookie == "" {
		t.Fatalf("old admin password must still log in as admin: %d", r.StatusCode)
	}
	z := authz{cookie: cookie}
	if b := bodyOf(r); !strings.Contains(b, `"must_change_password":true`) {
		t.Fatalf("login response must flag the forced change: %s", b)
	}
	if c := do(t, srv, "GET", "/api/servers", "", z).StatusCode; c != 403 {
		t.Fatalf("API before password change = %d, want 403", c)
	}
	me := do(t, srv, "GET", "/api/me", "", z)
	if me.StatusCode != 200 || !strings.Contains(bodyOf(me), `"must_change_password":true`) {
		t.Fatal("/api/me must stay reachable and report the flag")
	}
	if c := do(t, srv, "POST", "/api/password", `{"current":"wrong","new":"another-pass1"}`, z).StatusCode; c != 403 {
		t.Fatalf("wrong current password = %d", c)
	}
	if c := do(t, srv, "POST", "/api/password", `{"current":"pw","new":"short"}`, z).StatusCode; c != 400 {
		t.Fatalf("short password = %d", c)
	}
	if c := do(t, srv, "POST", "/api/password", `{"current":"pw","new":"another-pass1"}`, z).StatusCode; c != 200 {
		t.Fatalf("change = %d", c)
	}
	if c := do(t, srv, "GET", "/api/servers", "", z).StatusCode; c != 200 {
		t.Fatalf("API after change = %d", c)
	}
	// old password is dead
	api.auth.mu.Lock()
	api.auth.fails = map[string]failure{}
	api.auth.mu.Unlock()
	if r, _ := login(t, srv, "admin", "pw"); r.StatusCode != 401 {
		t.Fatalf("old password still works: %d", r.StatusCode)
	}
}

func TestBootstrapOnlyWhenNoUsers(t *testing.T) {
	e := setup(t)
	if err := e.store.BootstrapAdmin("first-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.store.CreateUser("bob", "bob-password", roleViewer, false); err != nil {
		t.Fatal(err)
	}
	// a second start must not recreate or reset admin
	if err := e.store.BootstrapAdmin("other-pass"); err != nil {
		t.Fatal(err)
	}
	n, _ := e.store.CountUsers()
	_, hash, _ := e.store.userWithHash("admin")
	if n != 2 || !verifyPassword(hash, "first-pass") || verifyPassword(hash, "other-pass") {
		t.Fatalf("bootstrap ran again: users=%d", n)
	}
}

// A database written by the previous release (schema v1, no users table) upgrades in place.
func TestUpgradeFromSchemaV1(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+dir+"/hub.db")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(schemaV1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO settings(key,value) VALUES('keep','"me"')`); err != nil {
		t.Fatal(err)
	}
	db.Close() // no schema_migrations table: looks like a pre-versioning database
	st, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var v string
	if ok, _ := st.GetSetting("keep", &v); !ok || v != "me" {
		t.Fatal("existing data lost in upgrade")
	}
	cfg := Config{AdminPassword: "old-admin-pw", AgentKey: "k", SessionSecret: "s", RetentionDays: 1}
	if _, err := newAuth(cfg, st); err != nil {
		t.Fatal(err)
	}
	u, hash, err := st.userWithHash("admin")
	if err != nil || u.Role != roleAdmin || !u.MustChange || !verifyPassword(hash, "old-admin-pw") {
		t.Fatalf("admin after upgrade: %+v %v", u, err)
	}
}

func TestPasswordHashFormat(t *testing.T) {
	h, err := hashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=") || strings.Contains(h, "correct horse") {
		t.Fatalf("hash = %s", h)
	}
	h2, _ := hashPassword("correct horse")
	if h == h2 {
		t.Fatal("salt must differ")
	}
	if !verifyPassword(h, "correct horse") || verifyPassword(h, "correct horsf") || verifyPassword("garbage", "x") || verifyPassword("", "") {
		t.Fatal("verify")
	}
}

// ---- sessions ----

func TestLogoutInvalidatesSessionServerSide(t *testing.T) {
	srv, api, _ := newAPI(t)
	mkUser(t, api, "sam", roleOperator)
	r, cookie := login(t, srv, "sam", "password-sam")
	if r.StatusCode != 200 {
		t.Fatalf("login = %d", r.StatusCode)
	}
	z := authz{cookie: cookie}
	if c := do(t, srv, "GET", "/api/servers", "", z).StatusCode; c != 200 {
		t.Fatalf("with session = %d", c)
	}
	var rows int
	_ = api.store.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&rows)
	var stored string
	_ = api.store.db.QueryRow(`SELECT id_hash FROM sessions`).Scan(&stored)
	if rows != 1 || stored == cookie || stored != sha256Hex(cookie) {
		t.Fatalf("session must be stored hashed: rows=%d stored=%s", rows, stored)
	}
	if c := do(t, srv, "POST", "/api/logout", "", z).StatusCode; c != 204 {
		t.Fatalf("logout = %d", c)
	}
	// the cookie value copied before logout no longer works
	if c := do(t, srv, "GET", "/api/servers", "", z).StatusCode; c != 401 {
		t.Fatalf("after logout = %d, want 401", c)
	}
	_ = api.store.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&rows)
	if rows != 0 {
		t.Fatal("session row must be deleted")
	}
}

func TestDisableUserKillsSessionsAndTokensAtOnce(t *testing.T) {
	srv, api, _ := newAPI(t)
	admin := authz{cookie: mkSession(t, api, mkUser(t, api, "boss", roleAdmin))}
	u := mkUser(t, api, "temp", roleOperator)
	sess := authz{cookie: mkSession(t, api, u)}
	tok := authz{bearer: mkToken(t, api, "temp", roleOperator)}
	for _, z := range []authz{sess, tok} {
		if c := do(t, srv, "GET", "/api/servers", "", z).StatusCode; c != 200 {
			t.Fatalf("before disable = %d", c)
		}
	}
	if c := do(t, srv, "PUT", "/api/users/"+itoa(u.ID), `{"disabled":true}`, admin).StatusCode; c != 200 {
		t.Fatalf("disable = %d", c)
	}
	for _, z := range []authz{sess, tok} {
		if c := do(t, srv, "GET", "/api/servers", "", z).StatusCode; c != 401 {
			t.Fatalf("after disable = %d, want 401", c)
		}
	}
	if r, _ := login(t, srv, "temp", "password-temp"); r.StatusCode != 401 {
		t.Fatalf("disabled user logs in: %d", r.StatusCode)
	}
	// enabling again restores the token (sessions stay gone)
	do(t, srv, "PUT", "/api/users/"+itoa(u.ID), `{"disabled":false}`, admin)
	if c := do(t, srv, "GET", "/api/servers", "", tok).StatusCode; c != 200 {
		t.Fatalf("token after re-enable = %d", c)
	}
	if c := do(t, srv, "GET", "/api/servers", "", sess).StatusCode; c != 401 {
		t.Fatalf("old session after re-enable = %d", c)
	}
}

func itoa(i int64) string { b, _ := json.Marshal(i); return string(b) }

func TestPasswordChangeEndsOtherSessions(t *testing.T) {
	srv, api, _ := newAPI(t)
	u := mkUser(t, api, "pat", roleViewer)
	a, b := authz{cookie: mkSession(t, api, u)}, authz{cookie: mkSession(t, api, u)}
	if c := do(t, srv, "POST", "/api/password", `{"current":"password-pat","new":"brand-new-pass"}`, a).StatusCode; c != 200 {
		t.Fatalf("change = %d", c)
	}
	if c := do(t, srv, "GET", "/api/servers", "", a).StatusCode; c != 200 {
		t.Fatalf("current session = %d", c)
	}
	if c := do(t, srv, "GET", "/api/servers", "", b).StatusCode; c != 401 {
		t.Fatalf("other session = %d", c)
	}
}

func TestAdminResetPasswordForcesChangeAndEndsSessions(t *testing.T) {
	srv, api, _ := newAPI(t)
	admin := authz{cookie: mkSession(t, api, mkUser(t, api, "boss", roleAdmin))}
	u := mkUser(t, api, "lost", roleViewer)
	s := authz{cookie: mkSession(t, api, u)}
	if c := do(t, srv, "POST", "/api/users/"+itoa(u.ID)+"/password", `{"password":"temporary-pass"}`, admin).StatusCode; c != 200 {
		t.Fatalf("reset = %d", c)
	}
	if c := do(t, srv, "GET", "/api/servers", "", s).StatusCode; c != 401 {
		t.Fatalf("old session = %d", c)
	}
	_, cookie := login(t, srv, "lost", "temporary-pass")
	if c := do(t, srv, "GET", "/api/servers", "", authz{cookie: cookie}).StatusCode; c != 403 {
		t.Fatalf("temporary password must force a change, got %d", c)
	}
}

func TestLastAdminIsProtected(t *testing.T) {
	srv, api, _ := newAPI(t)
	u := readyAdmin(t, api)
	z := authz{cookie: mkSession(t, api, u)}
	for _, tc := range []struct{ method, body string }{{"PUT", `{"disabled":true}`}, {"PUT", `{"role":"viewer"}`}, {"DELETE", ""}} {
		if c := do(t, srv, tc.method, "/api/users/"+itoa(u.ID), tc.body, z).StatusCode; c != 409 {
			t.Errorf("%s %s on the last admin = %d, want 409", tc.method, tc.body, c)
		}
	}
	other := mkUser(t, api, "second", roleAdmin)
	if c := do(t, srv, "PUT", "/api/users/"+itoa(other.ID), `{"role":"viewer"}`, z).StatusCode; c != 200 {
		t.Fatalf("demoting a non-last admin = %d", c)
	}
}

func TestUserValidationAndDuplicates(t *testing.T) {
	srv, api, _ := newAPI(t)
	u := readyAdmin(t, api)
	z := authz{cookie: mkSession(t, api, u)}
	post := func(body string) int { return do(t, srv, "POST", "/api/users", body, z).StatusCode }
	if c := post(`{"username":"nina","password":"long-enough-1","role":"viewer"}`); c != 200 {
		t.Fatalf("create = %d", c)
	}
	if c := post(`{"username":"NINA","password":"long-enough-1","role":"viewer"}`); c != 409 {
		t.Fatalf("duplicate (case-insensitive) = %d", c)
	}
	for _, bad := range []string{`{"username":"a b","password":"long-enough-1","role":"viewer"}`, `{"username":"x","password":"short","role":"viewer"}`, `{"username":"x","password":"long-enough-1","role":"root"}`} {
		if c := post(bad); c != 400 {
			t.Errorf("%s = %d, want 400", bad, c)
		}
	}
	// the new user has a forced change
	r, cookie := login(t, srv, "nina", "long-enough-1")
	if r.StatusCode != 200 || do(t, srv, "GET", "/api/servers", "", authz{cookie: cookie}).StatusCode != 403 {
		t.Fatal("new user must change the password first")
	}
}

// ---- API tokens ----

func TestAPITokens(t *testing.T) {
	srv, api, _ := newAPI(t)
	u := mkUser(t, api, "ops", roleOperator)
	z := authz{cookie: mkSession(t, api, u)}
	r := do(t, srv, "POST", "/api/tokens", `{"name":"ci","role_cap":"admin"}`, z)
	if r.StatusCode != 400 {
		t.Fatalf("cap above own role = %d", r.StatusCode)
	}
	r = do(t, srv, "POST", "/api/tokens", `{"name":"ci","role_cap":"viewer","expires_days":1}`, z)
	var made struct {
		Token string
		Info  APIToken
	}
	if r.StatusCode != 200 || json.NewDecoder(r.Body).Decode(&made) != nil || !strings.HasPrefix(made.Token, "pm2go_") {
		t.Fatalf("create token = %d %+v", r.StatusCode, made)
	}
	// hashed at rest: the secret appears nowhere in the table
	var hash, prefix string
	_ = api.store.db.QueryRow(`SELECT hash, prefix FROM api_tokens`).Scan(&hash, &prefix)
	if hash == made.Token || hash != sha256Hex(made.Token) || !strings.HasPrefix(made.Token, prefix) || len(prefix) >= len(made.Token) {
		t.Fatalf("token storage: hash=%s prefix=%s", hash, prefix)
	}
	// listing never shows the secret
	if b := bodyOf(do(t, srv, "GET", "/api/tokens", "", z)); strings.Contains(b, made.Token) || strings.Contains(b, hash) {
		t.Fatalf("token list leaks secret: %s", b)
	}
	bearer := authz{bearer: made.Token}
	if c := do(t, srv, "GET", "/api/servers", "", bearer).StatusCode; c != 200 {
		t.Fatalf("token read = %d", c)
	}
	// role_cap lowers the owner's operator role to viewer
	if c := do(t, srv, "POST", "/api/servers/box1/command", `{"action":"restart","target":"x"}`, bearer).StatusCode; c != 403 {
		t.Fatalf("capped token command = %d, want 403", c)
	}
	if c := do(t, srv, "POST", "/api/servers/box1/command", `{"action":"restart","target":"x"}`, z).StatusCode; c != 502 {
		t.Fatalf("owner session command = %d", c)
	}
	// tokens cannot manage credentials
	if c := do(t, srv, "POST", "/api/tokens", `{"name":"x"}`, bearer).StatusCode; c != 403 {
		t.Fatalf("token minting tokens = %d", c)
	}
	// last_used_at is recorded
	var used int64
	_ = api.store.db.QueryRow(`SELECT last_used_at FROM api_tokens`).Scan(&used)
	if used == 0 {
		t.Fatal("last_used_at not set")
	}
	// another user cannot revoke it, the owner can
	other := authz{cookie: mkSession(t, api, mkUser(t, api, "eve", roleViewer))}
	if c := do(t, srv, "DELETE", "/api/tokens/"+itoa(made.Info.ID), "", other).StatusCode; c != 404 {
		t.Fatalf("foreign revoke = %d", c)
	}
	if c := do(t, srv, "DELETE", "/api/tokens/"+itoa(made.Info.ID), "", z).StatusCode; c != 204 {
		t.Fatalf("revoke = %d", c)
	}
	if c := do(t, srv, "GET", "/api/servers", "", bearer).StatusCode; c != 401 {
		t.Fatalf("revoked token = %d, want 401", c)
	}
}

func TestExpiredTokenAndOldAdminPasswordBearerRejected(t *testing.T) {
	srv, api, cfg := newAPI(t)
	u := mkUser(t, api, "exp", roleAdmin)
	_, secret, _ := api.store.CreateToken(u.ID, "old", roleAdmin, time.Now().Add(-time.Minute).UnixMilli())
	if c := do(t, srv, "GET", "/api/servers", "", authz{bearer: secret}).StatusCode; c != 401 {
		t.Fatalf("expired token = %d", c)
	}
	if c := do(t, srv, "GET", "/api/servers", "", authz{bearer: cfg.AdminPassword}).StatusCode; c != 401 {
		t.Fatalf("admin password as bearer = %d, want 401 (replaced by API tokens)", c)
	}
	// the websocket honours tokens too
	tok := mkToken(t, api, "exp", roleViewer)
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/api/logs/ws?server=s", http.Header{"Authorization": {"Bearer " + tok}})
	if err != nil {
		t.Fatal(err)
	}
	ws.Close()
}

// ---- brute force ----

func TestLoginBackoffPerUserAndIP(t *testing.T) {
	srv, api, _ := newAPI(t)
	mkUser(t, api, "target", roleViewer)
	if r, _ := login(t, srv, "target", "bad-1"); r.StatusCode != 401 {
		t.Fatalf("first failure = %d", r.StatusCode)
	}
	// immediate retry, even with the right password, is refused
	if r, _ := login(t, srv, "target", "password-target"); r.StatusCode != 429 {
		t.Fatalf("retry during lockout = %d, want 429", r.StatusCode)
	}
	// guessing other usernames from the same IP is throttled as well
	if r, _ := login(t, srv, "someone-else", "x"); r.StatusCode != 429 {
		t.Fatalf("username spray from the same IP = %d, want 429", r.StatusCode)
	}
	api.auth.mu.Lock()
	for k, f := range api.auth.fails {
		f.last = time.Now().Add(-time.Minute)
		api.auth.fails[k] = f
	}
	api.auth.mu.Unlock()
	if r, _ := login(t, srv, "target", "password-target"); r.StatusCode != 200 {
		t.Fatalf("login after the wait = %d", r.StatusCode)
	}
	// the backoff grows with repeated failures
	for i := 0; i < 8; i++ {
		api.auth.mu.Lock()
		api.auth.fails = map[string]failure{}
		api.auth.mu.Unlock()
		login(t, srv, "target", "bad")
		api.auth.mu.Lock()
		api.auth.fail("127.0.0.1|target", "127.0.0.1")
		api.auth.mu.Unlock()
	}
	api.auth.mu.Lock()
	w := api.auth.wait("127.0.0.1|target")
	api.auth.mu.Unlock()
	if w < 2*time.Second {
		t.Fatalf("wait = %v", w)
	}
}

func TestUnknownAndDisabledUsersGetTheSameAnswer(t *testing.T) {
	srv, api, _ := newAPI(t)
	u := mkUser(t, api, "off", roleViewer)
	off := true
	api.store.UpdateUser(u.ID, nil, &off) // other admin (bootstrap) still exists
	r1, _ := login(t, srv, "off", "password-off")
	b1 := bodyOf(r1)
	api.auth.mu.Lock()
	api.auth.fails = map[string]failure{}
	api.auth.mu.Unlock()
	r2, _ := login(t, srv, "ghost", "whatever")
	if r1.StatusCode != 401 || r2.StatusCode != 401 || b1 != bodyOf(r2) {
		t.Fatalf("responses differ: %d %q vs %d", r1.StatusCode, b1, r2.StatusCode)
	}
}

// ---- audit ----

func TestAuditTrailCoversLoginsCommandsRulesAndUsersWithoutSecrets(t *testing.T) {
	srv, api, _ := newAPI(t)
	admin := mkUser(t, api, "root", roleAdmin)
	z := authz{cookie: mkSession(t, api, admin)}
	login(t, srv, "root", "WRONG-PASSWORD-123") // failure
	do(t, srv, "POST", "/api/servers/box1/command", `{"action":"restart","target":"api"}`, z)
	r := do(t, srv, "POST", "/api/alerts/rules", `{"name":"cpu high","enabled":true,"level":"warning","type":"proc_cpu","threshold":90,"duration_s":60}`, z)
	if r.StatusCode != 200 {
		t.Fatalf("add rule = %d %s", r.StatusCode, bodyOf(r))
	}
	do(t, srv, "PUT", "/api/settings/telegram", `{"enabled":false,"bot_token":"123:SUPER-SECRET-BOT-TOKEN","chats":[]}`, z)
	do(t, srv, "POST", "/api/users", `{"username":"newbie","password":"INITIAL-SECRET-PW","role":"viewer"}`, z)
	tr := do(t, srv, "POST", "/api/tokens", `{"name":"ci"}`, z)
	var made struct{ Token string }
	json.NewDecoder(tr.Body).Decode(&made)
	do(t, srv, "POST", "/api/password", `{"current":"WRONG-CURRENT","new":"NEW-SECRET-PASSWORD"}`, z)

	got := do(t, srv, "GET", "/api/audit?limit=200", "", z)
	raw := bodyOf(got)
	var rows []AuditEntry
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatal(err, raw)
	}
	has := func(action string, ok bool) bool {
		for _, e := range rows {
			if e.Action == action && e.Success == ok {
				return true
			}
		}
		return false
	}
	for _, w := range []struct {
		a  string
		ok bool
	}{{"login", false}, {"command", false}, {"rule.create", true}, {"settings.telegram", true}, {"user.create", true}, {"token.create", true}, {"password.change", false}} {
		if !has(w.a, w.ok) {
			t.Errorf("no audit row %s success=%v in %s", w.a, w.ok, raw)
		}
	}
	for _, secret := range []string{"WRONG-PASSWORD-123", "SUPER-SECRET-BOT-TOKEN", "INITIAL-SECRET-PW", "WRONG-CURRENT", "NEW-SECRET-PASSWORD", made.Token, cookieSecret(z)} {
		if secret != "" && strings.Contains(raw, secret) {
			t.Errorf("audit log contains secret %q", secret)
		}
	}
	for _, e := range rows {
		if e.IP == "" || e.TS == 0 {
			t.Errorf("row without ip or ts: %+v", e)
		}
	}
	// filters
	if b := bodyOf(do(t, srv, "GET", "/api/audit?failed=1", "", z)); strings.Contains(b, `"success":true`) {
		t.Errorf("failed filter returned successes: %s", b)
	}
	var cmds []AuditEntry
	json.NewDecoder(do(t, srv, "GET", "/api/audit?action=user.*", "", z).Body).Decode(&cmds)
	if len(cmds) == 0 {
		t.Error("prefix filter found nothing")
	}
}

func cookieSecret(z authz) string { return z.cookie }

func TestAuditRetentionPrune(t *testing.T) {
	e := setup(t)
	e.store.Audit(AuditEntry{TS: time.Now().Add(-48 * time.Hour).UnixMilli(), User: "x", IP: "i", Action: "old"})
	e.store.Audit(AuditEntry{User: "x", IP: "i", Action: "new"})
	if err := e.store.Prune(24 * time.Hour); err != nil {
		t.Fatal(err)
	}
	rows, _ := e.store.ListAudit(AuditFilter{})
	if len(rows) != 1 || rows[0].Action != "new" {
		t.Fatalf("rows after prune = %+v", rows)
	}
}

func TestLoginAndLogoutAreAudited(t *testing.T) {
	srv, api, _ := newAPI(t)
	mkUser(t, api, "lou", roleViewer)
	_, cookie := login(t, srv, "lou", "password-lou")
	do(t, srv, "POST", "/api/logout", "", authz{cookie: cookie})
	rows, _ := api.store.ListAudit(AuditFilter{User: "lou"})
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.Action] = r.Success
	}
	if !seen["login"] || !seen["logout"] {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestResetAdmin(t *testing.T) {
	e := setup(t)
	e.store.BootstrapAdmin("first-pass")
	u, _, _ := e.store.userWithHash("admin")
	raw, _, _ := e.store.CreateSession(u.ID, "i", "ua")
	off := true
	e.store.CreateUser("b", "b-password1", roleAdmin, false)
	e.store.UpdateUser(u.ID, nil, &off)
	if err := e.store.ResetAdmin("fresh-pass-1"); err != nil {
		t.Fatal(err)
	}
	u, hash, _ := e.store.userWithHash("admin")
	if u.Disabled || !u.MustChange || !verifyPassword(hash, "fresh-pass-1") {
		t.Fatalf("admin after reset: %+v", u)
	}
	if _, _, ok := e.store.LookupSession(raw); ok {
		t.Fatal("reset must end sessions")
	}
}

// ---- review follow-ups: bounded backoff map, cookie attributes, stalled websocket ----

func TestFailTableIsBounded(t *testing.T) {
	defer func(o int) { maxFailEntries = o }(maxFailEntries)
	maxFailEntries = 50
	_, api, _ := newAPI(t)
	for i := 0; i < 500; i++ { // distinct IPs that fail and never succeed
		api.auth.mu.Lock()
		api.auth.fail("10.0.0."+itoa(int64(i))+"|u", "10.0.0."+itoa(int64(i)))
		if n := len(api.auth.fails); n > maxFailEntries {
			api.auth.mu.Unlock()
			t.Fatalf("table grew to %d", n)
		}
		api.auth.mu.Unlock()
	}
	// the newest entries survive eviction
	api.auth.mu.Lock()
	_, ok := api.auth.fails["10.0.0.499"]
	api.auth.mu.Unlock()
	if !ok {
		t.Fatal("newest entry was evicted")
	}
}

func TestSessionCookieAttributesSetAndClear(t *testing.T) {
	_, api, _ := newAPI(t)
	for _, trust := range []bool{false, true} {
		api.auth.trust = trust
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		set := api.auth.sessionCookie(req, "v", time.Now().Add(time.Hour))
		rec := httptest.NewRecorder()
		api.auth.logout(rec, req)
		cleared := rec.Result().Cookies()
		if len(cleared) != 1 {
			t.Fatalf("logout cookies = %v", cleared)
		}
		c := cleared[0]
		if c.MaxAge >= 0 || c.Value != "" {
			t.Fatalf("not cleared: %+v", c)
		}
		if c.HttpOnly != set.HttpOnly || c.SameSite != set.SameSite || c.Secure != set.Secure || c.Path != set.Path {
			t.Fatalf("set %+v and clear %+v differ", set, c)
		}
		if !set.HttpOnly || set.SameSite != http.SameSiteStrictMode || set.Secure != trust {
			t.Fatalf("trust=%v attributes: %+v", trust, set)
		}
	}
}

func TestLogsWebsocketBacklogWriteHasDeadline(t *testing.T) {
	srv, api, _ := newAPI(t)
	api.wsTimeout = 300 * time.Millisecond
	mkUser(t, api, "wsv", roleViewer)
	tok := mkToken(t, api, "wsv", roleViewer)
	big := strings.Repeat("x", 60_000)
	var lines []common.LogLine
	for i := 0; i < 200; i++ {
		lines = append(lines, common.LogLine{TS: int64(i + 1), Name: "api", Stream: "out", Line: big})
	}
	api.reg.OnLogs("s1", lines)
	// a client that completes the handshake and then never reads
	u, _ := url.Parse(srv.URL)
	nc, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	if tc, ok := nc.(*net.TCPConn); ok {
		tc.SetReadBuffer(4096)
	}
	fmt.Fprintf(nc, "GET /api/logs/ws?server=s1&proc=api HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\nAuthorization: Bearer %s\r\n\r\n", u.Host, tok)
	// The server must give up on the stalled client and close the connection: after the
	// buffered bytes are drained the read ends with EOF, not with a timeout.
	time.Sleep(1200 * time.Millisecond) // stalled for 4x the write timeout
	nc.SetReadDeadline(time.Now().Add(4 * time.Second))
	buf := make([]byte, 1<<16)
	for {
		if _, err := nc.Read(buf); err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				t.Fatal("server kept the connection open to a client that does not read")
			}
			return
		}
	}
}
