# #93: CLI unit tests and testable command layer

Status: in review
Branch: test/pm2go-cli
PR: #111
Issue: https://github.com/rpsoftech/GolangServers/issues/93
Size: S-M | Lane: cli | Needs first: none
Hot files: `cli/commands.go`, `cli/client.go`

> The session working on #93 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-28, #93: `cli` exposes package variables `dial` (RPC transport), `spawn` (daemon launcher) and `spawnWait` as test hooks; start flags are collected in `startOpts`. `parseInterspersed` now splits at the first `--` before parsing, so flags placed after a positional and before `--` are honoured (previously they became positionals). No other behaviour change.
- 2026-11-28, #93, `test/pm2go-cli`: CLI unit tests plus in-process end-to-end test added, coverage of `servers/pm2go/cli` 85.9%, `go test -race` passes.

## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- (none yet)
