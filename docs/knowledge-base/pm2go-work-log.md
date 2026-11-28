# pm2go work log

Shared coordination file for every session working on pm2go. **Read this and `pm2go-dev-guide.md` first.** Do not re-read the whole codebase: the dev guide is the map and each issue lists the files it touches.

Rules for using this file:

1. Before you start an issue, set its row to `in progress` with your branch name (first commit of your branch).
2. Log every decision another session must know (new interface, changed protocol, renamed type, schema change) in **Decisions** and, if it changes how others work, in `pm2go-dev-guide.md`.
3. When you finish, set the row to `done` with the PR link, tick `pm2go-status.md`, and add a line to **Journal**.
4. Append only to Decisions and Journal. Edit your own status row. Keep rows short; details live in the issue.
5. If you find work that is out of scope, open a new issue and add a row here instead of expanding your PR.
6. Merge conflicts in this file are resolved by keeping both sides' lines.

## Baseline

- PR [#88](https://github.com/rpsoftech/GolangServers/pull/88) (`feat/pm2go-process-manager`) holds the base: process manager, agent, hub, dashboard, alerts, Telegram, Docker/Caddy files, docs. Until it merges, branch new work from `feat/pm2go-process-manager`; after it merges, branch from `main`.
- Internal use only, Linux only, single tenant (until the accounts issue). No billing, no tiers.
- Not verified on real infrastructure yet: see issue [#106](https://github.com/rpsoftech/GolangServers/issues/106).

## Issue board

Size: S small (hours), M medium (1 day), L large (several days). Lane: which area the work lives in (see **Order and parallelism**).

| Issue | Title | Size | Lane | Needs first | Hot files touched | Status | Branch / PR |
|-------|-------|------|------|-------------|-------------------|--------|-------------|
| [#89](https://github.com/rpsoftech/GolangServers/issues/89) | Hub schema versioning and migrations | S | hub | none | `hub/store.go`, new `hub/migrations.go` | open | |
| [#90](https://github.com/rpsoftech/GolangServers/issues/90) | Hub UI: split `app.js` into modules | S-M | hub UI | none | `hub/ui/*` | open | |
| [#91](https://github.com/rpsoftech/GolangServers/issues/91) | GitHub Actions: tests, binaries, Docker image | S-M | tooling | none | `.github/`, `deploy/Dockerfile` | in review | [#107](https://github.com/rpsoftech/GolangServers/pull/107) `feat/pm2go-ci-release` |
| [#92](https://github.com/rpsoftech/GolangServers/issues/92) | YAML ecosystem files, `ignore_watch` globs | S | cli/common | none | `common/ecosystem.go`, `daemon/watch.go` | open | |
| [#93](https://github.com/rpsoftech/GolangServers/issues/93) | CLI unit tests and testable command layer | S-M | cli | none | `cli/commands.go`, `cli/client.go` | open | |
| [#94](https://github.com/rpsoftech/GolangServers/issues/94) | `monit` terminal dashboard | M | cli | #93 (soft) | new `cli/monit.go` | open | |
| [#95](https://github.com/rpsoftech/GolangServers/issues/95) | Hub built-in TLS and secure defaults | S-M | hub | none | `hub/server.go`, `agent/agent.go` | open | |
| [#96](https://github.com/rpsoftech/GolangServers/issues/96) | Deployment tracking (git revision, deploy markers) | M | daemon+hub | #89, #90 | `common/types.go`, `hub/registry.go`, UI charts | open | |
| [#97](https://github.com/rpsoftech/GolangServers/issues/97) | `deploy` command (git + ssh) | M-L | cli | none (YAML #92 soft) | new `cli/deploy.go`, `common/deploy.go` | open | |
| [#98](https://github.com/rpsoftech/GolangServers/issues/98) | Shared listening socket and `wait_ready` | L | daemon | none | `daemon/proc.go`, `daemon/manager.go` | done | `feat/daemon-shared-socket`, [PR #112](https://github.com/rpsoftech/GolangServers/pull/112) |
| [#99](https://github.com/rpsoftech/GolangServers/issues/99) | Restart daemon without stopping children | L | daemon | #98 (order) | `daemon/proc.go`, `daemon/logfile.go`, `cli/startup.go` | open | |
| [#100](https://github.com/rpsoftech/GolangServers/issues/100) | More alert channels, silences, quiet hours | L | hub | #89, #90 | `hub/alerts.go`, `hub/telegram.go`, `hub/api.go` | open | |
| [#101](https://github.com/rpsoftech/GolangServers/issues/101) | Persistent log storage and search | L | hub | #89, #90 | `hub/registry.go`, `hub/api.go`, UI | open | |
| [#102](https://github.com/rpsoftech/GolangServers/issues/102) | User accounts, roles, API tokens, audit log | L | hub | #89, #90 | `hub/auth.go`, `hub/api.go` (every route), UI | open | |
| [#103](https://github.com/rpsoftech/GolangServers/issues/103) | Go SDK: custom metrics, actions, error reports | L | sdk+hub | #89, #90 | `common/proto.go`, `agent/agent.go`, `hub/*`, new `sdk/` | open | |
| [#104](https://github.com/rpsoftech/GolangServers/issues/104) | Transaction tracing | L | sdk+hub | #103 | `sdk/trace`, hub, UI | open | |
| [#105](https://github.com/rpsoftech/GolangServers/issues/105) | Remote CPU and heap profiling | L | sdk+hub | #103 | `sdk/profile.go`, hub, UI | open | |
| [#106](https://github.com/rpsoftech/GolangServers/issues/106) | Verify on real infrastructure (Docker, Caddy, Linux, Telegram) | M | human | none | docs only | open | |

Status values: `open`, `in progress`, `in review`, `done`, `blocked (reason)`.

## Order and parallelism

Goal: many sessions at once without stepping on each other. Pick work from different lanes, and never run two issues at once that touch the same hot file in big ways.

```
Wave 0 (do first, small):   #89 migrations   #90 UI split   #91 CI        (these three can run in parallel)
Wave 1 (parallel lanes):
  hub lane:      #102 accounts (alone in hub auth/api)  ->  then #100, #101, #96 in parallel  ->  #103 SDK  ->  #104, #105 in parallel
  daemon lane:   #98 shared socket  ->  #99 survive daemon restart
  cli lane:      #93 CLI tests  ->  #94 monit ;   #92 YAML ;   #97 deploy ;   #95 TLS
  human lane:    #106 real-infrastructure verification (can start as soon as someone has a Linux host)
```

Why this order:

- #89 and #90 remove the two biggest merge-conflict sources (no schema migrations, one giant `app.js`). Most hub issues need both.
- #102 changes the permission model of every route. Doing it before many new routes exist avoids retrofitting roles later. If it cannot go first, every new route must call a role check once #102 lands.
- #103 defines the SDK transport that #104 and #105 reuse. Keep `pm2go-sdk-protocol.md` current while building it.
- #98 and #99 both rewrite parts of `daemon/proc.go`. Run them one after the other, not together.
- #92 owns `common/ecosystem.go`; #97 adds `common/deploy.go` and only one field to `Ecosystem`.

Hot files and who may touch them heavily: `hub/ui/app.js` (#90 only, then modules), `hub/api.go` and `hub/auth.go` (#102), `hub/store.go` (#89), `daemon/proc.go` (#98, then #99), `common/proto.go` and `agent/agent.go` (#103, others additive only), `hub/alerts.go` (#100; #103, #104, #96 add one case each).

## Conventions for all sessions

- Branch names: `feat/<topic>` or `fix/<topic>`, plain words. No tool or AI names anywhere (commits, PRs, issues, comments, branch names). Commits are authored by the repository owner only, no co-author lines.
- Commit titles: `feat(pm2go): ...`, `fix(pm2go): ...`, `docs(pm2go): ...`, `test(pm2go): ...`.
- One PR per issue, titled `<type>(pm2go): <issue title>`, body links the issue (`Closes #N`). Base branch: `feat/pm2go-process-manager` until #88 merges, then `main`.
- Definition of done is in each issue. Short form: tests pass with `-race`, `gofmt` clean, `go vet` clean, Linux cross-build passes, docs and checklist updated, this file updated, UI checked in a browser.
- Commands and the manual end-to-end recipe: `pm2go-dev-guide.md`.
- Tests that bind sockets use a short `PMGO_HOME` under `/tmp` (macOS unix socket path limit).
- Always clean up test processes (`pgrep -fl pm2go` must be empty when you finish).

## Decisions

Append new entries at the bottom: date, issue, decision, reason.

- 2026-11-28, base: Linux-only production target; `daemon/sampler_other.go` (`ps`) exists only for development on macOS.
- 2026-11-28, base: hub uses standard library `net/http`, `gorilla/websocket`, pure-Go SQLite (`modernc.org/sqlite`, `CGO_ENABLED=0` builds) so the hub ships as one static binary and a small image.
- 2026-11-28, base: agents connect **outbound** to the hub; the hub never connects to hosts, so hosts need no inbound port.
- 2026-11-28, base: children's output is piped through the daemon (rotating writer plus agent hook). Consequence: children die with the daemon. Fixed by #99.
- 2026-11-28, base: dashboard is vanilla JS without a build step, strict CSP, DOM built with `h()` and `textContent` only. Keep it that way.
- 2026-11-28, base: alert model is rule per (type, level); several rules of one type give escalation; higher level covers lower for the same target; sustained thresholds via `duration_s`; reminders via `repeat_s`; resolve after 10 s of false.
- 2026-11-28, base: secrets (Telegram token) are write-only in the API (`token_set` flag). New secrets follow the same rule.
- 2026-11-28, base: no cluster-style port sharing yet; apps use `SO_REUSEPORT` themselves until #98.
- 2026-11-28, #98: `daemon/proc.go` now spawns through `Proc.spawn()` (`daemon/child.go`) and waits through `Proc.supervise()`; `loop` only does restart policy. One `child` = one OS process (cmd, waitCh, readiness). #99 should keep this split and adopt or re-attach children as `child` values.
- 2026-11-28, #98: shared sockets live in `Manager.lns` (`daemon/listener.go`), one per app name, held as a single dup'd `*os.File` (the `net.Listener` is closed right after `File()`). Children get it as fd 3 (`LISTEN_FDS=1`), readiness pipe is the next fd (`PMGO_READY_FD`). #99 must keep both open across a daemon restart (or hand them over) or reload and restart lose the port.
- 2026-11-28, #98: `Proc.Reload()` sends a request over `reloadReq` to the supervise loop (overlapping reload, only for apps with `listen`); `Manager.Reload` falls back to stop, start, wait when the loop is not running. `Manager.Reload` now returns an error (with infos) when an instance never becomes ready; the old instance stays up.
- 2026-11-28, #98: `LISTEN_PID` is not set (unknown before fork); readiness is a pipe only (no datagram socket variant).

## Journal

Append only. Newest at the bottom.

- 2026-11-28: PR #88 opened with the base implementation (daemon, CLI, agent, hub, dashboard, alerts, Telegram, Docker and Caddy files, docs). Race tests pass, Linux cross-build passes. Manual end-to-end run on macOS verified hub, agent, dashboard, live logs, issues, alert firing, remote restart. Not run on Linux, Docker or real Telegram.
- 2026-11-28: issues #89 to #106 created from the open items in `pm2go-status.md`, each with context, design, files, acceptance criteria and tests so sessions do not need to re-read the code.
- 2026-11-28: #98 shared listening socket (fd 3, `LISTEN_FDS`), `wait_ready` pipe protocol, overlapping zero-downtime `reload`, `sdk/runtime` helper, flags `--listen`, `--wait-ready`, `--listen-timeout`. Tests compile `daemon/testdata/sharedapp` and cover pid spread, zero-drop reload, never-ready kill, slow-ready wait, failed reload keeps old, fd leaks. Race tests, vet, gofmt and linux build pass. Tested on macOS only, not on Linux hardware.
- 2026-11-28, #91, feat/pm2go-ci-release: PR #107 adds `.github/workflows/pm2go.yml` (CI, release binaries, GHCR image). Not run: no Docker or actionlint locally; first CI run is the first real image build. Tag `pm2go-v*` publishes after merge.

## Handoff template (copy into your Journal entry when you stop mid-way)

```
- <date>, #<issue>, <branch>: stopped at <step>. Done: <list>. Next: <list>. Gotchas: <list>. Tests run: <commands and result>.
```
