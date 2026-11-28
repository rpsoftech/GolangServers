# #102: User accounts, roles, API tokens, audit log

Status: in review
Branch: feat/hub-accounts
PR: 
Issue: https://github.com/rpsoftech/GolangServers/issues/102
Size: L | Lane: hub | Needs first: #89, #90
Hot files: `hub/auth.go`, `hub/api.go` (every route), UI

> The session working on #102 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

- 2026-11-29: server-side sessions (hashed id in `sessions`), not token-version cookies. Why: logout and disable must take effect at once; `session_secret` is now unused.
- 2026-11-29: argon2id (x/crypto already a direct dependency). API tokens `pm2go_...`, SHA-256 at rest, shown once. Why: tokens are high-entropy, so a fast hash is fine.
- 2026-11-29: every new hub route needs a role: register with `viewer(...)`/`operator(...)`/`admin(...)` in `API.Handler` and add it to `routeMatrix` in `hub/accounts_test.go` (a test fails otherwise). Issue #100 (silences): create = operator. Why: acceptance matrix.
- 2026-11-29: `newAuth(cfg, store)` now takes the store and returns an error; `a.audit(r, action, target, detail, ok)` writes audit rows from handlers (never pass secrets). UI: `can('admin')`, `ME`, `registerNav(href, label, minRole)` in `core.js`.
- 2026-11-29: `GET /api/settings/telegram` is admin only; `settings/info` hides `agent_key` below admin. Bearer admin password no longer works (use API tokens).

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.


## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- [ ] multi-user accounts, roles, audit log (#102) (SSO phase 2 stays open: add a separate line for it)

## Verification (2026-11-29)

`go test -race -count=1 ./servers/pm2go/...` ok, `gofmt -l` empty, `go vet` clean, linux CGO_ENABLED=0 build ok. Browser check on a local hub: bootstrap login as `admin` forces the password change, Users, Audit and Account pages, token creation (shown once), viewer sees no mutation controls. SSO phase 2 not done. Behaviour docs: `pm2go-accounts.md`.
