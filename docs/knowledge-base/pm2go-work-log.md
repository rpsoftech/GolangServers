# pm2go work log

Shared coordination file for every session working on pm2go. **Read this and `pm2go-dev-guide.md` first.** Do not re-read the whole codebase: the dev guide is the map and each issue lists the files it touches.

**This file is static and owned by the coordinator. Sessions do not edit it.** It exists so that parallel sessions never produce merge conflicts after each other's merges. Where to write instead:

| What | Where |
|------|-------|
| Your status, branch, PR, decisions, notes, handoff | your own file `docs/knowledge-base/pm2go-work/<NNN>-<slug>.md` (one per issue, listed in the board below) |
| New behaviour documentation | a **new** file `docs/knowledge-base/pm2go-<topic>.md` (for example `pm2go-sdk-protocol.md`) |
| Corrections to existing docs (`pm2go.md`, `pm2go-dev-guide.md`, `pm2go-deploy.md`) | edit the wrong text **in place**; never append to lists or tables, never add rows |
| Ticking `pm2go-status.md` | do not edit it; write the lines to tick under "Status-doc ticks" in your issue file, the coordinator ticks them at the end |
| Anything else you discover that is out of scope | open a new GitHub issue and mention it in your issue file |

Why: every PR used to append to the same journal and table rows in this file, so each merge made all other open PRs conflict. One file per issue means two PRs never touch the same lines of shared docs.

See the live board with `docs/knowledge-base/pm2go-work/board.sh` (reads the `Status:`, `Branch:` and `PR:` lines of every issue file).

## Baseline

- PR [#88](https://github.com/rpsoftech/GolangServers/pull/88) (`feat/pm2go-process-manager`) holds the base: process manager, agent, hub, dashboard, alerts, Telegram, Docker/Caddy files, docs. Until it merges, branch new work from `feat/pm2go-process-manager`; after it merges, branch from `main`.
- Internal use only, Linux only, single tenant (until the accounts issue). No billing, no tiers.
- Not verified on real infrastructure yet: see issue [#106](https://github.com/rpsoftech/GolangServers/issues/106).

## Issue board

Size: S small (hours), M medium (1 day), L large (several days). Lane: which area the work lives in (see **Order and parallelism**).

| Issue | Title | Size | Lane | Needs first | Hot files touched | Detail file |
|-------|-------|------|------|-------------|-------------------|-------------|
| [#89](https://github.com/rpsoftech/GolangServers/issues/89) | Hub schema versioning and migrations | S | hub | none | `hub/store.go`, new `hub/migrations.go` | [`089-hub-schema-versioning-and-migrations.md`](pm2go-work/089-hub-schema-versioning-and-migrations.md) |
| [#90](https://github.com/rpsoftech/GolangServers/issues/90) | Hub UI: split `app.js` into modules | S-M | hub UI | none | `hub/ui/*` | [`090-hub-ui-split-app-js-into-modules.md`](pm2go-work/090-hub-ui-split-app-js-into-modules.md) |
| [#91](https://github.com/rpsoftech/GolangServers/issues/91) | GitHub Actions: tests, binaries, Docker image | S-M | tooling | none | `.github/`, `deploy/Dockerfile` | [`091-github-actions-tests-binaries-docker-ima.md`](pm2go-work/091-github-actions-tests-binaries-docker-ima.md) |
| [#92](https://github.com/rpsoftech/GolangServers/issues/92) | YAML ecosystem files, `ignore_watch` globs | S | cli/common | none | `common/ecosystem.go`, `daemon/watch.go` | [`092-yaml-ecosystem-files-ignore-watch-globs.md`](pm2go-work/092-yaml-ecosystem-files-ignore-watch-globs.md) |
| [#93](https://github.com/rpsoftech/GolangServers/issues/93) | CLI unit tests and testable command layer | S-M | cli | none | `cli/commands.go`, `cli/client.go` | [`093-cli-unit-tests-and-testable-command-laye.md`](pm2go-work/093-cli-unit-tests-and-testable-command-laye.md) |
| [#94](https://github.com/rpsoftech/GolangServers/issues/94) | `monit` terminal dashboard | M | cli | #93 (soft) | new `cli/monit.go` | [`094-monit-terminal-dashboard.md`](pm2go-work/094-monit-terminal-dashboard.md) |
| [#95](https://github.com/rpsoftech/GolangServers/issues/95) | Hub built-in TLS and secure defaults | S-M | hub | none | `hub/server.go`, `agent/agent.go` | [`095-hub-built-in-tls-and-secure-defaults.md`](pm2go-work/095-hub-built-in-tls-and-secure-defaults.md) |
| [#96](https://github.com/rpsoftech/GolangServers/issues/96) | Deployment tracking (git revision, deploy markers) | M | daemon+hub | #89, #90 | `common/types.go`, `hub/registry.go`, UI charts | [`096-deployment-tracking-git-revision-deploy.md`](pm2go-work/096-deployment-tracking-git-revision-deploy.md) |
| [#97](https://github.com/rpsoftech/GolangServers/issues/97) | `deploy` command (git + ssh) | M-L | cli | none (YAML #92 soft) | new `cli/deploy.go`, `common/deploy.go` | [`097-deploy-command-git-ssh.md`](pm2go-work/097-deploy-command-git-ssh.md) |
| [#98](https://github.com/rpsoftech/GolangServers/issues/98) | Shared listening socket and `wait_ready` | L | daemon | none | `daemon/proc.go`, `daemon/manager.go` | [`098-shared-listening-socket-and-wait-ready.md`](pm2go-work/098-shared-listening-socket-and-wait-ready.md) |
| [#99](https://github.com/rpsoftech/GolangServers/issues/99) | Restart daemon without stopping children | L | daemon | #98 (order) | `daemon/proc.go`, `daemon/logfile.go`, `cli/startup.go` | [`099-restart-daemon-without-stopping-children.md`](pm2go-work/099-restart-daemon-without-stopping-children.md) |
| [#100](https://github.com/rpsoftech/GolangServers/issues/100) | More alert channels, silences, quiet hours | L | hub | #89, #90 | `hub/alerts.go`, `hub/telegram.go`, `hub/api.go` | [`100-more-alert-channels-silences-quiet-hours.md`](pm2go-work/100-more-alert-channels-silences-quiet-hours.md) |
| [#101](https://github.com/rpsoftech/GolangServers/issues/101) | Persistent log storage and search | L | hub | #89, #90 | `hub/registry.go`, `hub/api.go`, UI | [`101-persistent-log-storage-and-search.md`](pm2go-work/101-persistent-log-storage-and-search.md) |
| [#102](https://github.com/rpsoftech/GolangServers/issues/102) | User accounts, roles, API tokens, audit log | L | hub | #89, #90 | `hub/auth.go`, `hub/api.go` (every route), UI | [`102-user-accounts-roles-api-tokens-audit-log.md`](pm2go-work/102-user-accounts-roles-api-tokens-audit-log.md) |
| [#103](https://github.com/rpsoftech/GolangServers/issues/103) | Go SDK: custom metrics, actions, error reports | L | sdk+hub | #89, #90 | `common/proto.go`, `agent/agent.go`, `hub/*`, new `sdk/` | [`103-go-sdk-custom-metrics-actions-error-repo.md`](pm2go-work/103-go-sdk-custom-metrics-actions-error-repo.md) |
| [#104](https://github.com/rpsoftech/GolangServers/issues/104) | Transaction tracing | L | sdk+hub | #103 | `sdk/trace`, hub, UI | [`104-transaction-tracing.md`](pm2go-work/104-transaction-tracing.md) |
| [#105](https://github.com/rpsoftech/GolangServers/issues/105) | Remote CPU and heap profiling | L | sdk+hub | #103 | `sdk/profile.go`, hub, UI | [`105-remote-cpu-and-heap-profiling.md`](pm2go-work/105-remote-cpu-and-heap-profiling.md) |
| [#106](https://github.com/rpsoftech/GolangServers/issues/106) | Verify on real infrastructure (Docker, Caddy, Linux, Telegram) | M | human | none | docs only | [`106-verify-on-real-infrastructure-docker-cad.md`](pm2go-work/106-verify-on-real-infrastructure-docker-cad.md) |

Status values (in the `Status:` line of each issue file): `open`, `in progress`, `in review`, `done`, `blocked (reason)`.

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
- Definition of done is in each issue. Short form: tests pass with `-race`, `gofmt` clean, `go vet` clean, Linux cross-build passes, docs updated (new topics in new files, see the table at the top), your issue file updated, UI checked in a browser.
- Commands and the manual end-to-end recipe: `pm2go-dev-guide.md`.
- Tests that bind sockets use a short `PMGO_HOME` under `/tmp` (macOS unix socket path limit).
- Always clean up test processes (`pgrep -fl pm2go` must be empty when you finish).

## Decisions

Baseline decisions from the first implementation. New decisions go in the `## Decisions` section of your own issue file, not here.

- 2026-11-28, base: Linux-only production target; `daemon/sampler_other.go` (`ps`) exists only for development on macOS.
- 2026-11-28, base: hub uses standard library `net/http`, `gorilla/websocket`, pure-Go SQLite (`modernc.org/sqlite`, `CGO_ENABLED=0` builds) so the hub ships as one static binary and a small image.
- 2026-11-28, base: agents connect **outbound** to the hub; the hub never connects to hosts, so hosts need no inbound port.
- 2026-11-28, base: children's output is piped through the daemon (rotating writer plus agent hook). Consequence: children die with the daemon. Fixed by #99.
- 2026-11-28, base: dashboard is vanilla JS without a build step, strict CSP, DOM built with `h()` and `textContent` only. Keep it that way.
- 2026-11-28, base: alert model is rule per (type, level); several rules of one type give escalation; higher level covers lower for the same target; sustained thresholds via `duration_s`; reminders via `repeat_s`; resolve after 10 s of false.
- 2026-11-28, base: secrets (Telegram token) are write-only in the API (`token_set` flag). New secrets follow the same rule.
- 2026-11-28, base: no cluster-style port sharing yet; apps use `SO_REUSEPORT` themselves until #98.
