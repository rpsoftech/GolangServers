# #91: GitHub Actions: tests, binaries, Docker image

Status: done
Branch: feat/pm2go-ci-release
PR: #107
Issue: https://github.com/rpsoftech/GolangServers/issues/91
Size: S-M | Lane: tooling | Needs first: none
Hot files: `.github/`, `deploy/Dockerfile`

> The session working on #91 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-28, #91, feat/pm2go-ci-release: PR #107 adds `.github/workflows/pm2go.yml` (CI, release binaries, GHCR image). Not run: no Docker or actionlint locally; first CI run is the first real image build. Tag `pm2go-v*` publishes after merge.

## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- (none yet)
