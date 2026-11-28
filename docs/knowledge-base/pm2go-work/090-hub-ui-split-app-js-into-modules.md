# #90: Hub UI: split `app.js` into modules

Status: done
Branch: feat/hub-ui-modules
PR: #109
Issue: https://github.com/rpsoftech/GolangServers/issues/90
Size: S-M | Lane: hub UI | Needs first: none
Hot files: `hub/ui/*`

> The session working on #90 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

## Log

Progress, what was verified, what was not, gotchas. Newest at the bottom.

- 2026-11-28, #90, PR #109: dashboard split into classic scripts (`core.js`, `charts.js`, `login.js`, `proc-table.js`, `page-*.js`, boot-only `app.js`) with `registerPage(pattern, fn)` and `registerNav(href, label)`. New page = one `page-x.js` plus one `<script>` line in `index.html`. Checked every page in a browser; `TestUIServesIndexAndEveryScript` added.

## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- (none yet)
