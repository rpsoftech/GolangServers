# Project journal (coordinator only)

Project-level events: PR lifecycle of the integration branch, planning, releases. Sessions working on an issue write to their own issue file instead.

## Log

- 2026-11-28: PR #88 opened with the base implementation (daemon, CLI, agent, hub, dashboard, alerts, Telegram, Docker and Caddy files, docs). Race tests pass, Linux cross-build passes. Manual end-to-end run on macOS verified hub, agent, dashboard, live logs, issues, alert firing, remote restart. Not run on Linux, Docker or real Telegram.
- 2026-11-28: issues #89 to #106 created from the open items in `pm2go-status.md`, each with context, design, files, acceptance criteria and tests so sessions do not need to re-read the code.
- 2026-11-28: shared docs restructured so parallel PRs cannot conflict: static work log, one file per issue under `pm2go-work/`, no appends to shared lists.
- 2026-11-29: PRs #107 to #112 merged into the integration branch; sub-branches, worktrees and sessions cleaned up; code graph and knowledge graph added under `docs/knowledge-base/graph/`; assessment in `pm2go-assessment.md`.
- 2026-11-29: review fixes on the integration branch: `ready_timeout` exits count toward restart-loop alerts, `Registry.OnLogs` takes the registry lock once per batch, agent keeps a bounded event backlog while offline and replays it after hello (hello is now always the first message). Review items on `hub/auth.go` (bounded login-failure map, `trustProxy` out of a global, logout cookie attributes) were handed to the #102 session.
