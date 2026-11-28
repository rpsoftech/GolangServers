# #92: YAML ecosystem files, `ignore_watch` globs

Status: review
Branch: feat/ecosystem-yaml
PR: (see issue #92)
Issue: https://github.com/rpsoftech/GolangServers/issues/92
Size: S | Lane: cli/common | Needs first: none
Hot files: `common/ecosystem.go`, `daemon/watch.go`

> The session working on #92 edits **only this file** for status, decisions and notes
> (see `README.md` in this folder). Append at the bottom of a section. Never edit another issue's file.

## Decisions

Anything another session must know: new interfaces, protocol or schema changes, renamed types, rules.
Format: `- <date>: <decision>. Why: <reason>.`

- 2026-11-29: `common.LoadEcosystemWarn(path, env)` returns apps plus unknown-key warnings; `LoadEcosystem` keeps its signature. `common.IsEcosystemFile(target)` detects by extension. Why: other callers stay unchanged; CLI diff stays small.
- 2026-11-29: Unknown keys are derived from the json tags of `Ecosystem` and `AppConfig` via reflect, so new fields (for example `deploy` from #97) are known automatically. Why: no second key list to maintain.
- 2026-11-29: go.mod diff is one line: `go.yaml.in/yaml/v3` loses `// indirect` (left in place, no `go mod tidy`). Why: avoid touching other dependencies; note in PR that this triggers the release workflow.
- 2026-11-29: `ignore_watch` pattern with an inner `/` is anchored to cwd (gitignore style); bare names match at any depth; `ignored()` now takes `isDir`.

## Log

- 2026-11-29: Implemented; `go test -race -count=1 ./servers/pm2go/...`, gofmt, vet and linux CGO_ENABLED=0 build pass. Tests: JSON/YAML equivalence, env profiles, invalid YAML, unknown keys, glob table, watcher ignoring `**/*.log` and `tmp/`.

Progress, what was verified, what was not, gotchas. Newest at the bottom.


## Handoff

Fill this only if you stop before the work is done: stopped at, done, next, gotchas, commands run.

## Status-doc ticks to apply (the coordinator ticks `pm2go-status.md` later)

- Tick: YAML ecosystem files, richer `ignore_watch` globs (#92)
