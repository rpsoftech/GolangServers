# pm2go assessment: done, pending, improvements

Snapshot after PRs #107 to #112 merged into the integration branch `feat/pm2go-process-manager` (PR #88 to `main` still open). Status per issue: `pm2go-work/board.sh`. Graphs: `graph/GRAPH.md`.

## Done

| Area | What exists | Verified how |
|------|-------------|--------------|
| Process manager | start, stop, restart, reload, delete, scale, list, describe, logs, flush, save, resurrect; restart policy with backoff and `stop_exit_codes`; instances; `watch`, `cron_restart`, `max_memory_restart`; size-rotated gzip logs; JSON ecosystem files with env profiles; systemd unit generation; shared listening socket and `wait_ready` for zero-downtime reload (#98) | race tests on macOS |
| Agent | outbound WebSocket to the hub, snapshots, events, log batches, remote commands | hub tests with a fake agent, manual run |
| Hub | SQLite store with versioned migrations (#89), session and agent-key auth, REST API, live log WebSocket, issues grouping, optional built-in TLS and `--hub-ca` (#95) | tests, manual run (TLS autocert not verified) |
| Dashboard | modular vanilla JS UI (#90): servers, server detail charts, process detail with live logs, issues, alerts with rule editor, settings | manual browser check |
| Alerts | leveled rules, sustained thresholds, reminders, resolved messages, escalation, 13 default rules; Telegram with per-chat minimum level | fake-clock tests, fake Telegram server |
| Deployment | Dockerfile, compose, Caddy files, trusted proxy, `/healthz`, CI workflow and release workflow (#91) | CI ran the workflow on its PR; image never run by hand |
| Quality | CLI tests with 85.9% coverage (#93), hub and daemon tests | `go test -race` |
| Coordination | issues #89 to #106, per-issue work files, dev guide, code graph and knowledge graph | n/a |

## Pending

Not started (issue, size, depends on):

| Issue | Feature | Size | Needs first |
|-------|---------|------|-------------|
| #102 | accounts, roles, API tokens, audit log | L | none (do before new routes) |
| #100 | email, Slack, webhook channels, silences, quiet hours | L | #102 preferred |
| #101 | persistent log storage and search | L | none |
| #96 | deployment tracking, deploy markers on charts | M | none |
| #103 | Go SDK: custom metrics, actions, error reports | L | none |
| #104 | transaction tracing | L | #103 |
| #105 | remote CPU and heap profiling | L | #103 |
| #99 | restart the daemon without stopping children | L | #98 (done) |
| #92 | YAML ecosystem files, `ignore_watch` globs | S | none |
| #94 | `monit` terminal dashboard | M | #93 (done) |
| #97 | `deploy` command (git + ssh) | M-L | none |
| #106 | verify on real infrastructure (Docker, Caddy, Linux, Telegram) | M | a person with the platform |

Merge step: PR #88 (integration branch to `main`) once, then close the done issues (#89, #90, #91, #93, #95, #98 carry `Closes` lines in the PR body).

Not verified anywhere yet: the Docker image and compose stack run by hand, Caddy block on the platform, `/proc` numbers against `top`, `pm2go startup` after a reboot, real Telegram delivery, ACME against Let's Encrypt.

## Improvements beyond the issues (suggested, not yet tracked)

Priority: P1 do soon, P2 worthwhile, P3 later.

**Security**
- P1: `start` from the dashboard runs an arbitrary command as the daemon user. Gate it behind the admin role (#102), audit it, and consider disabling it by default with a hub setting.
- P1: per-host agent keys (revocable, named) instead of one shared key; show last seen per key.
- P2: rate limit all API routes, not only login; CSRF token for state-changing routes in addition to `SameSite=Strict`; security headers review.
- P2: encrypt stored secrets (Telegram token, SMTP password, webhook secrets) with a key from the environment.
- P3: signed commands from hub to agent, so a compromised proxy cannot issue commands.

**Reliability**
- P1: agent buffers events and a bounded number of snapshots while offline and replays them (today events during a disconnect are lost, so a crash during a network blip is never alerted).
- P1: `Registry.OnLogs` holds the registry lock while fanning out to subscribers; move fan-out off the lock.
- P2: batch SQLite writes and add a write queue metric; document the single-writer limit and the point where to move to Postgres.
- P2: hub self-metrics: Prometheus `/metrics` on hub and daemon (queue depths, dropped log lines, agent count, alert evaluation time).
- P2: protocol version in `Hello`; hub warns when agent and hub versions differ.
- P3: hub high availability (warm standby with shared storage).

**Process management**
- P1: health checks (HTTP, TCP, command) that restart or alert on failure, not only on process exit.
- P2: cgroups v2 limits (memory, CPU) per app and `uid`/`gid` drop; env file support; start order and dependencies between apps; log timestamps and merged logs.
- P2: self-update (`pm2go update`) reusing the repository's `utility/updater`; `pm2go version`.
- P3: Windows and macOS production support (today development only).

**Metrics and alerts**
- P2: disk per mount, network per interface, per-process file descriptors, threads and IO; configurable sample interval.
- P2: alert acknowledge and assign in the dashboard, runbook link per rule, flapping detection, maintenance mode per server (overlaps #100 silences).
- P3: Telegram inline buttons (acknowledge, restart) through a bot webhook.

**Dashboard**
- P2: tags and groups for servers, search and favourites, mobile layout, light theme, CSV export of charts.
- P2: automated UI tests (browser based) in CI; today the UI is checked by hand.
- P3: per-process environment and config diff between restarts.

**Testing and delivery**
- P1: Linux-only sampler tests against a fake `/proc` root, and a CI matrix leg that runs the daemon tests on Linux (today the `/proc` code is only compiled, not run).
- P1: one integration test that starts hub, agent and daemon in a single Go test (the manual run, automated).
- P2: soak and load tests (many processes, chatty logs) with hub CPU and database growth recorded; chaos test (`kill -9` the daemon, the hub, the network).
- P2: fuzz tests for `parseInterspersed`, the ecosystem loader and the error-line fingerprint.
- P2: packages (deb, rpm), install script with checksum verification, release notes automation.
- P3: Helm chart or systemd unit for the hub next to the compose file.

**Docs**
- P2: troubleshooting and runbook page (agent not connecting, alerts not arriving, high hub CPU, database too big).
- P2: operator guide with screenshots of the dashboard.
