package hub

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"time"
)

// Sessions are server-side: the cookie holds 32 random bytes, the database
// holds only their SHA-256, so a leaked database cannot be replayed. Logout,
// disabling a user or changing a password deletes rows and takes effect at once.

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func randToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// CreateSession returns the raw cookie value and its expiry.
func (s *Store) CreateSession(userID int64, ip, userAgent string) (string, time.Time, error) {
	raw := randToken(32)
	now := time.Now()
	exp := now.Add(sessionTTL)
	if len(userAgent) > 200 {
		userAgent = userAgent[:200]
	}
	_, err := s.db.Exec(`INSERT INTO sessions(id_hash,user_id,created_at,expires_at,ip,user_agent) VALUES(?,?,?,?,?,?)`,
		sha256Hex(raw), userID, now.UnixMilli(), exp.UnixMilli(), ip, userAgent)
	return raw, exp, err
}

// LookupSession resolves a cookie value to an enabled user.
func (s *Store) LookupSession(raw string) (User, string, bool) {
	if raw == "" {
		return User{}, "", false
	}
	id := sha256Hex(raw)
	u, err := scanUser(s.db.QueryRow(`SELECT u.id, u.username, u.role, u.disabled, u.must_change_password, u.created_at, u.last_login_at
		FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id_hash=? AND s.expires_at>? AND u.disabled=0`, id, time.Now().UnixMilli()))
	if err != nil {
		return User{}, "", false
	}
	return u, id, true
}

func (s *Store) DeleteSession(idHash string) {
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE id_hash=?`, idHash)
}

// DeleteUserSessions ends every session of a user except keep (may be empty).
func (s *Store) DeleteUserSessions(userID int64, keep string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id=? AND id_hash<>?`, userID, keep)
	return err
}

// ---- API tokens ----

const tokenPrefix = "pm2go_"

// APIToken is the stored form of a token. The secret itself is shown once at
// creation and only its SHA-256 is kept.
type APIToken struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	User       string `json:"user,omitempty"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	RoleCap    string `json:"role_cap"`
	CreatedAt  int64  `json:"created_at"`
	LastUsedAt int64  `json:"last_used_at"`
	ExpiresAt  int64  `json:"expires_at"`
}

// CreateToken returns the stored token and the secret (shown once).
func (s *Store) CreateToken(userID int64, name, roleCap string, expiresAt int64) (APIToken, string, error) {
	raw := tokenPrefix + randToken(32)
	now := time.Now().UnixMilli()
	prefix := raw[:len(tokenPrefix)+4]
	res, err := s.db.Exec(`INSERT INTO api_tokens(user_id,name,prefix,hash,role_cap,created_at,expires_at) VALUES(?,?,?,?,?,?,?)`,
		userID, name, prefix, sha256Hex(raw), roleCap, now, expiresAt)
	if err != nil {
		return APIToken{}, "", err
	}
	id, _ := res.LastInsertId()
	return APIToken{ID: id, UserID: userID, Name: name, Prefix: prefix, RoleCap: roleCap, CreatedAt: now, ExpiresAt: expiresAt}, raw, nil
}

// LookupToken resolves a bearer secret to its token and an enabled owner.
func (s *Store) LookupToken(raw string) (User, APIToken, bool) {
	if len(raw) < len(tokenPrefix)+8 || raw[:len(tokenPrefix)] != tokenPrefix {
		return User{}, APIToken{}, false
	}
	now := time.Now().UnixMilli()
	var t APIToken
	var u User
	var dis, must int
	err := s.db.QueryRow(`SELECT t.id, t.user_id, t.name, t.prefix, t.role_cap, t.created_at, t.last_used_at, t.expires_at,
		u.id, u.username, u.role, u.disabled, u.must_change_password, u.created_at, u.last_login_at
		FROM api_tokens t JOIN users u ON u.id=t.user_id WHERE t.hash=? AND u.disabled=0 AND (t.expires_at=0 OR t.expires_at>?)`,
		sha256Hex(raw), now).Scan(&t.ID, &t.UserID, &t.Name, &t.Prefix, &t.RoleCap, &t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt,
		&u.ID, &u.Username, &u.Role, &dis, &must, &u.CreatedAt, &u.LastLoginAt)
	if err != nil {
		return User{}, APIToken{}, false
	}
	u.Disabled, u.MustChange = dis != 0, must != 0
	if t.LastUsedAt < now-60_000 { // at most one write a minute per token
		_, _ = s.db.Exec(`UPDATE api_tokens SET last_used_at=? WHERE id=?`, now, t.ID)
	}
	return u, t, true
}

// ListTokens returns one user's tokens, or everyone's when userID is 0.
func (s *Store) ListTokens(userID int64) ([]APIToken, error) {
	q := `SELECT t.id, t.user_id, u.username, t.name, t.prefix, t.role_cap, t.created_at, t.last_used_at, t.expires_at
		FROM api_tokens t JOIN users u ON u.id=t.user_id`
	args := []any{}
	if userID != 0 {
		q += ` WHERE t.user_id=?`
		args = append(args, userID)
	}
	rows, err := s.db.Query(q+` ORDER BY t.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []APIToken{}
	for rows.Next() {
		var t APIToken
		if rows.Scan(&t.ID, &t.UserID, &t.User, &t.Name, &t.Prefix, &t.RoleCap, &t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt) == nil {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

// GetToken loads one token (for the audit trail and ownership checks).
func (s *Store) GetToken(id int64) (APIToken, error) {
	var t APIToken
	err := s.db.QueryRow(`SELECT t.id, t.user_id, u.username, t.name, t.prefix, t.role_cap, t.created_at, t.last_used_at, t.expires_at
		FROM api_tokens t JOIN users u ON u.id=t.user_id WHERE t.id=?`, id).
		Scan(&t.ID, &t.UserID, &t.User, &t.Name, &t.Prefix, &t.RoleCap, &t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt)
	if err == sql.ErrNoRows {
		return t, errNotFound
	}
	return t, err
}

func (s *Store) DeleteToken(id int64) error {
	_, err := s.db.Exec(`DELETE FROM api_tokens WHERE id=?`, id)
	return err
}
