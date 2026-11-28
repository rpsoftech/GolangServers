# #95: Hub built-in TLS and secure defaults

Status: done
Branch: feat/hub-tls
PR: #110
Issue: https://github.com/rpsoftech/GolangServers/issues/95
Size: S-M | Lane: hub | Needs first: none
Hot files: `hub/server.go`, `agent/agent.go`

> The session working on #95 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-28, #95, `feat/hub-tls`: done, PR #110. Certificate files with reload, autocert, `--hub-ca`, docs and tests done locally; autocert not verified against Let's Encrypt.

## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- [x] TLS inside the hub: certificate files with reload, autocert, `link --hub-ca` (#95); autocert not yet checked against Let's Encrypt
