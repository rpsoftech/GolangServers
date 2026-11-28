# Project journal (coordinator only)

Project-level events: PR lifecycle of the integration branch, planning, releases. Sessions working on an issue write to their own issue file instead.

## Log

- 2026-11-28: PR #88 opened with the base implementation (daemon, CLI, agent, hub, dashboard, alerts, Telegram, Docker and Caddy files, docs). Race tests pass, Linux cross-build passes. Manual end-to-end run on macOS verified hub, agent, dashboard, live logs, issues, alert firing, remote restart. Not run on Linux, Docker or real Telegram.
- 2026-11-28: issues #89 to #106 created from the open items in `pm2go-status.md`, each with context, design, files, acceptance criteria and tests so sessions do not need to re-read the code.
- 2026-11-28: shared docs restructured so parallel PRs cannot conflict: static work log, one file per issue under `pm2go-work/`, no appends to shared lists.
