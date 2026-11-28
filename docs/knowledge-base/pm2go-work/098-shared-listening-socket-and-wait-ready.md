# #98: Shared listening socket and `wait_ready`

Status: in review
Branch: feat/daemon-shared-socket
PR: #112
Issue: https://github.com/rpsoftech/GolangServers/issues/98
Size: L | Lane: daemon | Needs first: none
Hot files: `daemon/proc.go`, `daemon/manager.go`

> The session working on #98 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-28, #98: `daemon/proc.go` now spawns through `Proc.spawn()` (`daemon/child.go`) and waits through `Proc.supervise()`; `loop` only does restart policy. One `child` = one OS process (cmd, waitCh, readiness). #99 should keep this split and adopt or re-attach children as `child` values.
- 2026-11-28, #98: shared sockets live in `Manager.lns` (`daemon/listener.go`), one per app name, held as a single dup'd `*os.File` (the `net.Listener` is closed right after `File()`). Children get it as fd 3 (`LISTEN_FDS=1`), readiness pipe is the next fd (`PMGO_READY_FD`). #99 must keep both open across a daemon restart (or hand them over) or reload and restart lose the port.
- 2026-11-28, #98: `Proc.Reload()` sends a request over `reloadReq` to the supervise loop (overlapping reload, only for apps with `listen`); `Manager.Reload` falls back to stop, start, wait when the loop is not running. `Manager.Reload` now returns an error (with infos) when an instance never becomes ready; the old instance stays up.
- 2026-11-28, #98: `LISTEN_PID` is not set (unknown before fork); readiness is a pipe only (no datagram socket variant).
- 2026-11-28: #98 shared listening socket (fd 3, `LISTEN_FDS`), `wait_ready` pipe protocol, overlapping zero-downtime `reload`, `sdk/runtime` helper, flags `--listen`, `--wait-ready`, `--listen-timeout`. Tests compile `daemon/testdata/sharedapp` and cover pid spread, zero-drop reload, never-ready kill, slow-ready wait, failed reload keeps old, fd leaks. Race tests, vet, gofmt and linux build pass. Tested on macOS only, not on Linux hardware.

## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- (none yet)
