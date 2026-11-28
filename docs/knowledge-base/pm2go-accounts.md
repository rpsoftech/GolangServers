# pm2go hub accounts, roles, API tokens and audit log

Phase 1 of issue #102: local accounts. SSO (OIDC) is phase 2 and not built. Code: `hub/auth.go` (middleware, login backoff), `hub/users.go` (accounts, argon2id), `hub/sessions.go` (sessions and API tokens), `hub/audit.go`, `hub/api_accounts.go` (handlers), migration 2 in `hub/migrations.go`. Tests: `hub/accounts_test.go`.

## Roles

| Role | Can do |
|------|--------|
| viewer | read everything except secrets (agent key, Telegram settings, users, audit log) |
| operator | viewer plus process commands (restart, reload, stop, scale, flush, start, save) and issue status |
| admin | everything: alert rules, Telegram settings, delete a process, users, audit log, the agent key |

A request's effective role is the user's role, lowered by the API token's `role_cap` for token requests. The check is `auth.requireRole(min, handler)`; the UI hides what a role cannot use but never decides anything.

## Route to role

| Route | Minimum role |
|-------|--------------|
| `GET /`, `/healthz`, `POST /api/login`, `POST /api/logout` | none |
| `GET /agent/ws` | agent key (not a user) |
| `GET /api/me` | any signed-in user, also while a password change is pending |
| `POST /api/password` | any session (not a token), also while a change is pending |
| `GET /api/servers`, `/servers/{name}`, `/servers/{name}/metrics`, `/servers/{name}/logs`, `/logs/ws`, `/events`, `/issues` | viewer |
| `GET /api/alerts/rules`, `/alerts/active`, `/alerts/history` | viewer |
| `GET /api/settings/info` | viewer (`agent_key` only in the admin response) |
| `POST /api/servers/{name}/command` | operator; action `delete` needs admin |
| `POST /api/issues/status` | operator |
| `POST /api/alerts/rules`, `PUT`/`DELETE /api/alerts/rules/{id}` | admin |
| `GET`/`PUT /api/settings/telegram`, `POST /api/settings/telegram/test` | admin |
| `GET /api/users` | admin |
| `POST /api/users`, `PUT`/`DELETE /api/users/{id}`, `POST /api/users/{id}/password`, `POST /api/users/{id}/revoke-sessions` | admin, session only |
| `GET`/`POST /api/tokens`, `DELETE /api/tokens/{id}`, `POST /api/sessions/revoke-all` | any user, session only (admins also list `?all=1` and revoke any token) |
| `GET /api/audit` | admin |

"Session only" routes refuse API tokens, so a stolen token cannot mint more tokens, change passwords or create users. `TestRouteMatrixCoversEveryRoute` fails when a route in `api.go` is missing from the matrix, and `TestRoleMatrix` runs every route as no user, viewer, operator and admin, by session and by token.

## Bootstrap and upgrade

On start, when the `users` table is empty, the hub creates user `admin` (role admin) with the password in `config.json` (`admin_password`) and sets `must_change_password`. Existing installs keep working: the old password logs in as `admin`, and the UI asks for a new one before anything else (every other route answers 403 `password change required`). After that `admin_password` is never read again. `pm2go hub --reset-admin` gives `admin` a new random password (creates or re-enables it, ends its sessions) for a lost password. The old "Bearer admin password" API access is gone; use API tokens.

## Passwords

argon2id (`golang.org/x/crypto/argon2`, m=64 MiB, t=1, p=4, 16-byte salt), stored as a PHC string with its own parameters so they can change later. At most 4 hashes run at once. A login for an unknown user verifies a dummy hash so timing does not reveal which names exist. Minimum 8 characters, maximum 256. New and reset passwords are temporary: the user must change them at next sign-in. Changing a password ends the user's other sessions.

## Sessions (decision: server-side)

The cookie `pm2go_session` holds 32 random bytes; the `sessions` table stores only their SHA-256, with expiry (7 days), IP and user agent. Logout deletes the row, disabling a user deletes all their rows, password change and reset delete the others. Cookie flags are unchanged (`HttpOnly`, `SameSite=Strict`, `Secure` over HTTPS). `session_secret` in `config.json` is no longer used. Expired rows are pruned hourly.

## API tokens

`Authorization: Bearer pm2go_<43 random chars>`. Created under Account, shown once, stored as SHA-256 plus a short display prefix. Fields: name, `role_cap` (at most the creator's role), optional expiry, `last_used_at` (updated at most once a minute). A token stops working when it is revoked, expired, or its owner is disabled or deleted.

## Login backoff

2 seconds per recent failure, up to 12 seconds, tracked per IP+username (targeted guessing) and per IP (guessing across usernames). Both answer 429 while waiting, even for the right password. Entries expire after 10 minutes idle. The IP comes from `clientIP` (honours trust-proxy).

## Audit log

Table `audit(id, ts, user, ip, action, target, detail, success)`, pruned after `retention_days` with the other data (raise it if you need a longer trail). `GET /api/audit?user=&action=&failed=1&before=<id>&limit=` (admin); `action` may end in `*` (`user.*`). Actions: `login` (success and failure; the name is logged only when it is a real account, otherwise `unknown`), `logout`, `command`, `issue.status`, `rule.create|update|delete`, `settings.telegram`, `settings.telegram.test`, `user.create|update|delete|password_reset|revoke_sessions`, `password.change`, `session.revoke_all`, `token.create|revoke`. Detail holds names, ids and flags, never passwords, tokens, the bot token or session ids (tested). Requests made with an API token are tagged `(api token)`.

## Not done

- Silences (issue #100) do not exist yet; when they land, `POST`/`DELETE` silence routes need operator (create) and the matrix entries.
- Phase 2: OIDC login with `state` and PKCE, email-domain allow-list, auto-provision as viewer.
- No per-user rate limit on API token use, no TOTP.
