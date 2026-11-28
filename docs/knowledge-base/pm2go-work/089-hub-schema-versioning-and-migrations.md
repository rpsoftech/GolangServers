# #89: Hub schema versioning and migrations

Status: done
Branch: feat/hub-schema-migrations
PR: #108
Issue: https://github.com/rpsoftech/GolangServers/issues/89
Size: S | Lane: hub | Needs first: none
Hot files: `hub/store.go`, new `hub/migrations.go`

> The session working on #89 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-28, #89, `feat/hub-schema-migrations`: done, PR #108. `schema_migrations` table plus append-only `migrations` list in `hub/migrations.go`; add a migration instead of editing old ones. Tests and Linux build pass.

## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- (none yet)
