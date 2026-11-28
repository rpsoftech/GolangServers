# pm2go work folder

One file per GitHub issue (`<NNN>-<slug>.md`), plus `000-project.md` (coordinator). This is how parallel sessions share state without merge conflicts.

## Rules

1. Work on issue #N: edit **only** `<NNN>-*.md` for status, decisions, notes and handoff. Set `Status:`, `Branch:`, `PR:` at the start (first commit) and at the end.
2. Never edit another issue's file, `pm2go-work-log.md`, or `pm2go-status.md`. If you need something from another issue, write it in your own `Decisions` section and mention the issue number; the other session reads it when it starts.
3. New behaviour docs go in a new `docs/knowledge-base/pm2go-<topic>.md`. Existing docs get in-place corrections only (no appended rows or list items).
4. Hot code files: keep changes small, put new code in new files, rebase before opening the PR. Adding a dashboard page means adding one `page-<name>.js` file plus (until the loader is automatic) one `<script>` line in `hub/ui/index.html`, keep that line in alphabetical order among the `page-*` lines.
5. When done: `Status: done`, `PR:` link, list the checklist lines to tick under "Status-doc ticks".

## See the board

```bash
docs/knowledge-base/pm2go-work/board.sh
```

Prints issue, status, branch and PR for every file. For live PR state use `gh pr list --search "pm2go"`.

## New issue file

Copy `TEMPLATE.md` to `<NNN>-<slug>.md`, fill the header, nothing else needs to be touched.
