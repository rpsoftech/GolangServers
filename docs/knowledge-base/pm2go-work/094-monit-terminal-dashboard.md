# #94: `monit` terminal dashboard

Status: done
Branch: feat/cli-monit
PR: (see pull request that closes #94)
Issue: https://github.com/rpsoftech/GolangServers/issues/94
Size: M | Lane: cli | Needs first: #93 (soft)
Hot files: new `cli/monit.go`

> The session working on #94 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

- 2026-11-29: new additive RPC `system` (`common.MethodSystem`) returns `Manager.System()`. Why: host summary line. Old daemons answer `unknown method`; monit hides host figures then.
- 2026-11-29: `golang.org/x/term` promoted from indirect to direct by dropping `// indirect` in `go.mod` (the only go.mod change; any go.mod change makes the release workflow release every component once it reaches main). Why: raw mode and terminal size.
- 2026-11-29: monit never auto-spawns the daemon and acts on numeric process ids. Why: a dashboard should not start a daemon; names can map to several instances.
- 2026-11-29: SIGWINCH registration lives in `cli/monit_unix.go` / `monit_windows.go` (build tags) because the integration branch vets Windows. Other CLI code: do not use `syscall.SIGWINCH` directly.

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-29: implemented in `cli/monit.go`; docs in `pm2go-monit.md`. Tests, `gofmt`, `go vet` (linux, darwin, windows), linux CGO-off build pass.
- Manual run against a real daemon in a pty at 80x24, 200x40: list, bars, logs, `e` toggle, `/` filter, `d` + `y` delete (processes disappeared from `pm2go ls`), `ctrl-c` leaves the alternate screen and shows the cursor. Host figures read 0 on macOS (the `ps` fallback sampler has no host data); Linux not run here.


## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- [ ] `monit` terminal dashboard (#94)
