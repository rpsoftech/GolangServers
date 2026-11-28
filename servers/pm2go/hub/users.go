package hub

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// Roles, lowest to highest. A higher role can do everything a lower one can.
const (
	roleViewer   = "viewer"
	roleOperator = "operator"
	roleAdmin    = "admin"
)

var roleRank = map[string]int{roleViewer: 1, roleOperator: 2, roleAdmin: 3}

func validRole(r string) bool { return roleRank[r] > 0 }

// minRole returns the lower of two roles.
func minRole(a, b string) string {
	if roleRank[a] <= roleRank[b] {
		return a
	}
	return b
}

var (
	errUserExists = errors.New("username already exists")
	errLastAdmin  = errors.New("cannot remove or disable the last enabled admin")
	errNotFound   = errors.New("not found")
	usernameRe    = regexp.MustCompile(`^[A-Za-z0-9._@-]{1,64}$`)
)

const minPasswordLen = 8

func checkPassword(pw string) error {
	if len(pw) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	if len(pw) > 256 {
		return errors.New("password too long")
	}
	return nil
}

// User is an account. The password hash is never part of this type.
type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	Disabled    bool   `json:"disabled"`
	MustChange  bool   `json:"must_change_password"`
	CreatedAt   int64  `json:"created_at"`
	LastLoginAt int64  `json:"last_login_at"`
}

const userCols = `id, username, role, disabled, must_change_password, created_at, last_login_at`

type scanner interface{ Scan(...any) error }

func scanUser(s scanner, extra ...any) (User, error) {
	var u User
	var dis, must int
	err := s.Scan(append([]any{&u.ID, &u.Username, &u.Role, &dis, &must, &u.CreatedAt, &u.LastLoginAt}, extra...)...)
	u.Disabled, u.MustChange = dis != 0, must != 0
	return u, err
}

// ---- password hashing: argon2id, PHC string format ----

// Parameters follow the RFC 9106 low-memory recommendation. They are stored in
// each hash, so changing them later keeps old hashes verifiable.
var (
	argonMem     uint32 = 64 * 1024 // KiB
	argonTime    uint32 = 1
	argonThreads uint8  = 4
	// hashSem bounds concurrent hashing so a burst of logins cannot exhaust memory.
	hashSem = make(chan struct{}, 4)
)

func hashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hashSem <- struct{}{}
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMem, argonThreads, 32)
	<-hashSem
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMem, argonTime, argonThreads,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

func verifyPassword(encoded, pw string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || m == 0 || m > 256*1024 || t == 0 || t > 10 || p == 0 {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[4])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(want) == 0 {
		return false
	}
	hashSem <- struct{}{}
	got := argon2.IDKey([]byte(pw), salt, t, m, p, uint32(len(want)))
	<-hashSem
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyHash is verified against when the username does not exist, so a missing
// user costs the same time as a wrong password.
var dummyHash string

func init() { dummyHash, _ = hashPassword("pm2go-dummy-password") }

// ---- user store ----

func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(username, password, role string, mustChange bool) (User, error) {
	if !usernameRe.MatchString(username) {
		return User{}, errors.New("username must be 1-64 characters: letters, digits and . _ @ -")
	}
	if !validRole(role) {
		return User{}, errors.New("role must be viewer, operator or admin")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}
	res, err := s.db.Exec(`INSERT INTO users(username,password_hash,role,must_change_password,created_at) VALUES(?,?,?,?,?)`,
		username, hash, role, b2i(mustChange), time.Now().UnixMilli())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return User{}, errUserExists
		}
		return User{}, err
	}
	id, _ := res.LastInsertId()
	return s.GetUser(id)
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) GetUser(id int64) (User, error) {
	u, err := scanUser(s.db.QueryRow(`SELECT `+userCols+` FROM users WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return u, errNotFound
	}
	return u, err
}

// userWithHash loads a user by name for login.
func (s *Store) userWithHash(username string) (User, string, error) {
	var hash string
	u, err := scanUser(s.db.QueryRow(`SELECT `+userCols+`, password_hash FROM users WHERE username=?`, username), &hash)
	if err == sql.ErrNoRows {
		return u, "", errNotFound
	}
	return u, hash, err
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT ` + userCols + ` FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		if u, err := scanUser(rows); err == nil {
			out = append(out, u)
		}
	}
	return out, rows.Err()
}

func (s *Store) TouchLogin(id int64) {
	_, _ = s.db.Exec(`UPDATE users SET last_login_at=? WHERE id=?`, time.Now().UnixMilli(), id)
}

// otherAdmins counts enabled admins besides id.
func (s *Store) otherAdmins(id int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role='admin' AND disabled=0 AND id<>?`, id).Scan(&n)
	return n, err
}

// UpdateUser changes the role and/or the disabled flag. Disabling a user ends
// all of their sessions at once; their API tokens stop working because token
// lookup requires an enabled user.
func (s *Store) UpdateUser(id int64, role *string, disabled *bool) (User, error) {
	s.userMu.Lock()
	defer s.userMu.Unlock()
	u, err := s.GetUser(id)
	if err != nil {
		return u, err
	}
	newRole, newDis := u.Role, u.Disabled
	if role != nil {
		if !validRole(*role) {
			return u, errors.New("role must be viewer, operator or admin")
		}
		newRole = *role
	}
	if disabled != nil {
		newDis = *disabled
	}
	if u.Role == roleAdmin && !u.Disabled && (newRole != roleAdmin || newDis) {
		if n, err := s.otherAdmins(id); err != nil {
			return u, err
		} else if n == 0 {
			return u, errLastAdmin
		}
	}
	if _, err := s.db.Exec(`UPDATE users SET role=?, disabled=? WHERE id=?`, newRole, b2i(newDis), id); err != nil {
		return u, err
	}
	if newDis {
		_ = s.DeleteUserSessions(id, "")
	}
	return s.GetUser(id)
}

// SetPassword stores a new password. All sessions of the user end except
// keepSession (the one that made the change; empty ends them all).
func (s *Store) SetPassword(id int64, password string, mustChange bool, keepSession string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`UPDATE users SET password_hash=?, must_change_password=? WHERE id=?`, hash, b2i(mustChange), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return s.DeleteUserSessions(id, keepSession)
}

func (s *Store) DeleteUser(id int64) error {
	s.userMu.Lock()
	defer s.userMu.Unlock()
	u, err := s.GetUser(id)
	if err != nil {
		return err
	}
	if u.Role == roleAdmin && !u.Disabled {
		if n, err := s.otherAdmins(id); err != nil {
			return err
		} else if n == 0 {
			return errLastAdmin
		}
	}
	for _, q := range []string{`DELETE FROM sessions WHERE user_id=?`, `DELETE FROM api_tokens WHERE user_id=?`, `DELETE FROM users WHERE id=?`} {
		if _, err := s.db.Exec(q, id); err != nil {
			return err
		}
	}
	return nil
}

// BootstrapAdmin creates user "admin" from the config password when there are
// no users yet, so installs that predate accounts keep working. The password
// must be changed at first login.
func (s *Store) BootstrapAdmin(password string) error {
	n, err := s.CountUsers()
	if err != nil || n > 0 {
		return err
	}
	if password == "" {
		return errors.New("admin_password is empty in config.json")
	}
	_, err = s.CreateUser("admin", password, roleAdmin, true)
	return err
}

// ResetAdmin gives the user "admin" a new password (creating or re-enabling it
// as an admin) and ends its sessions. Used by `pm2go hub --reset-admin`.
func (s *Store) ResetAdmin(password string) error {
	u, _, err := s.userWithHash("admin")
	if err == errNotFound {
		_, err = s.CreateUser("admin", password, roleAdmin, true)
		return err
	}
	if err != nil {
		return err
	}
	admin, off := roleAdmin, false
	if _, err := s.UpdateUser(u.ID, &admin, &off); err != nil {
		return err
	}
	return s.SetPassword(u.ID, password, true, "")
}
