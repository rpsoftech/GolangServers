# #99: Restart daemon without stopping children

Status: in progress
Branch: feat/daemon-survive-restart
PR: 
Issue: https://github.com/rpsoftech/GolangServers/issues/99
Size: L | Lane: daemon | Needs first: #98 (order)
Hot files: `daemon/proc.go`, `daemon/logfile.go`, `cli/startup.go`

> The session working on #99 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-29: implemented logs-as-files, state file, adoption, unit change. Race tests, vet, gofmt and the linux build pass on macOS. Linux-only code (`procinfo_linux.go`, `exitwatch_linux.go`: `/proc` start time, `pidfd_open`, `pidfd_getfd`) was compiled with `CGO_ENABLED=0 GOOS=linux` and vetted but not run. Gotcha: on macOS `File.Sync` is slow, so a test must wait until the state file holds the pids before it kills the daemon.


## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- [x] daemon restart without stopping children (#99)
