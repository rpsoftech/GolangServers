package hub

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// audit records an action by the authenticated principal of r.
func (a *API) audit(r *http.Request, action, target, detail string, ok bool) {
	user := ""
	if p := principalOf(r); p != nil {
		user = p.User.Username
		if p.Via == "token" {
			detail = strings.TrimSpace(detail + " (api token)")
		}
	}
	a.store.Audit(AuditEntry{User: user, IP: clientIP(r), Action: action, Target: target, Detail: detail, Success: ok})
}

func ruleDetail(r Rule) string {
	return fmt.Sprintf("%s type=%s level=%s enabled=%v", r.Name, r.Type, r.Level, r.Enabled)
}

type meView struct {
	OK bool `json:"ok"`
	User
	EffectiveRole string `json:"effective_role"`
	Via           string `json:"via"`
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	p := principalOf(r)
	writeJSON(w, meView{OK: true, User: p.User, EffectiveRole: p.Role, Via: p.Via})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &b) {
		return
	}
	if len(b.Username) > 64 || len(b.Password) > 256 {
		httpErr(w, http.StatusUnauthorized, "wrong username or password")
		return
	}
	if u, ok := a.auth.login(w, r, b.Username, b.Password); ok {
		writeJSON(w, map[string]any{"ok": true, "must_change_password": u.MustChange})
	}
}

// changePassword changes the caller's own password and ends their other sessions.
func (a *API) changePassword(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if !readJSON(w, r, &b) {
		return
	}
	p := principalOf(r)
	if err := checkPassword(b.New); err != nil {
		httpErr(w, 400, err.Error())
		return
	}
	_, hash, err := a.store.userWithHash(p.User.Username)
	if err != nil || !verifyPassword(hash, b.Current) {
		a.audit(r, "password.change", p.User.Username, "wrong current password", false)
		httpErr(w, http.StatusForbidden, "current password is wrong")
		return
	}
	if b.New == b.Current {
		httpErr(w, 400, "new password must differ from the current one")
		return
	}
	if err := a.store.SetPassword(p.User.ID, b.New, false, p.session); err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	a.audit(r, "password.change", p.User.Username, "", true)
	writeJSON(w, map[string]bool{"ok": true})
}

// ---- users (admin) ----

func userID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpErr(w, 400, "bad id")
		return 0, false
	}
	return id, true
}

func userErr(w http.ResponseWriter, err error) {
	switch err {
	case errNotFound:
		httpErr(w, 404, err.Error())
	case errUserExists, errLastAdmin:
		httpErr(w, 409, err.Error())
	default:
		httpErr(w, 400, err.Error())
	}
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	us, err := a.store.ListUsers()
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	writeJSON(w, us)
}

// createUser adds an account; the user must change the given password at first login.
func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var b struct{ Username, Password, Role string }
	if !readJSON(w, r, &b) {
		return
	}
	if err := checkPassword(b.Password); err != nil {
		httpErr(w, 400, err.Error())
		return
	}
	u, err := a.store.CreateUser(b.Username, b.Password, b.Role, true)
	if err != nil {
		a.audit(r, "user.create", b.Username, err.Error(), false)
		userErr(w, err)
		return
	}
	a.audit(r, "user.create", u.Username, "role="+u.Role, true)
	writeJSON(w, u)
}

func (a *API) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	var b struct {
		Role     *string `json:"role"`
		Disabled *bool   `json:"disabled"`
	}
	if !readJSON(w, r, &b) {
		return
	}
	u, err := a.store.UpdateUser(id, b.Role, b.Disabled)
	if err != nil {
		a.audit(r, "user.update", strconv.FormatInt(id, 10), err.Error(), false)
		userErr(w, err)
		return
	}
	a.audit(r, "user.update", u.Username, fmt.Sprintf("role=%s disabled=%v", u.Role, u.Disabled), true)
	writeJSON(w, u)
}

func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	u, err := a.store.GetUser(id)
	if err == nil {
		err = a.store.DeleteUser(id)
	}
	if err != nil {
		a.audit(r, "user.delete", strconv.FormatInt(id, 10), err.Error(), false)
		userErr(w, err)
		return
	}
	a.audit(r, "user.delete", u.Username, "", true)
	w.WriteHeader(204)
}

// resetUserPassword sets a temporary password; the user must change it at next login and loses all sessions.
func (a *API) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	var b struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &b) {
		return
	}
	if err := checkPassword(b.Password); err != nil {
		httpErr(w, 400, err.Error())
		return
	}
	u, err := a.store.GetUser(id)
	if err == nil {
		err = a.store.SetPassword(id, b.Password, true, "")
	}
	if err != nil {
		a.audit(r, "user.password_reset", strconv.FormatInt(id, 10), err.Error(), false)
		userErr(w, err)
		return
	}
	a.audit(r, "user.password_reset", u.Username, "", true)
	writeJSON(w, map[string]bool{"ok": true})
}

func (a *API) revokeUserSessions(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	u, err := a.store.GetUser(id)
	if err != nil {
		userErr(w, err)
		return
	}
	if err := a.store.DeleteUserSessions(id, ""); err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	a.audit(r, "user.revoke_sessions", u.Username, "", true)
	writeJSON(w, map[string]bool{"ok": true})
}

// revokeMySessions ends every session of the caller except the current one.
func (a *API) revokeMySessions(w http.ResponseWriter, r *http.Request) {
	p := principalOf(r)
	if err := a.store.DeleteUserSessions(p.User.ID, p.session); err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	a.audit(r, "session.revoke_all", p.User.Username, "", true)
	writeJSON(w, map[string]bool{"ok": true})
}

// ---- API tokens ----

// listTokens returns the caller's tokens; admins can pass ?all=1 for everyone's.
func (a *API) listTokens(w http.ResponseWriter, r *http.Request) {
	p := principalOf(r)
	uid := p.User.ID
	if r.URL.Query().Get("all") == "1" && p.Role == roleAdmin {
		uid = 0
	}
	ts, err := a.store.ListTokens(uid)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	writeJSON(w, ts)
}

// createToken returns the secret once. role_cap cannot exceed the caller's role.
func (a *API) createToken(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Name        string `json:"name"`
		RoleCap     string `json:"role_cap"`
		ExpiresDays int    `json:"expires_days"`
	}
	if !readJSON(w, r, &b) {
		return
	}
	p := principalOf(r)
	if b.RoleCap == "" {
		b.RoleCap = p.Role
	}
	if len(b.Name) == 0 || len(b.Name) > 80 {
		httpErr(w, 400, "name must be 1-80 characters")
		return
	}
	if !validRole(b.RoleCap) || roleRank[b.RoleCap] > roleRank[p.Role] {
		httpErr(w, 400, "role_cap must be viewer, operator or admin and not above your own role")
		return
	}
	if b.ExpiresDays < 0 || b.ExpiresDays > 3650 {
		httpErr(w, 400, "expires_days must be between 0 (never) and 3650")
		return
	}
	var exp int64
	if b.ExpiresDays > 0 {
		exp = time.Now().Add(time.Duration(b.ExpiresDays) * 24 * time.Hour).UnixMilli()
	}
	t, secret, err := a.store.CreateToken(p.User.ID, b.Name, b.RoleCap, exp)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	a.audit(r, "token.create", t.Prefix, fmt.Sprintf("name=%s role_cap=%s", t.Name, t.RoleCap), true)
	writeJSON(w, map[string]any{"token": secret, "info": t})
}

// deleteToken revokes a token. Owners revoke their own; admins revoke any.
func (a *API) deleteToken(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	p := principalOf(r)
	t, err := a.store.GetToken(id)
	if err != nil || (t.UserID != p.User.ID && p.Role != roleAdmin) {
		httpErr(w, 404, "token not found") // do not reveal other users' token ids
		return
	}
	if err := a.store.DeleteToken(id); err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	a.audit(r, "token.revoke", t.Prefix, "name="+t.Name+" owner="+t.User, true)
	w.WriteHeader(204)
}

// ---- audit log (admin) ----

func (a *API) auditLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	out, err := a.store.ListAudit(AuditFilter{User: q.Get("user"), Action: q.Get("action"), OnlyFailed: q.Get("failed") == "1", Before: before, Limit: limit})
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	writeJSON(w, out)
}
