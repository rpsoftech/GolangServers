# pm2go knowledge base

Internal process manager modelled on PM2, plus a web dashboard modelled on PM2 Plus. Go, Linux only in production, internal use only (no billing, no tiers, single tenant). Feature research is in `pm2-go-feature-analysis.md`, deployment (Docker, Caddy, systemd) in `pm2go-deploy.md`, and the done/pending checklist in `pm2go-status.md`.

One binary, `servers/pm2go`, runs in three roles:

- **CLI**: `pm2go start|stop|restart|reload|delete|scale|ls|describe|logs|flush|save|resurrect|link|unlink|startup|kill`.
- **Daemon** (`pm2go daemon`, auto-spawned by the CLI, or run by systemd with `--resurrect`): supervises processes and hosts the agent.
- **Hub** (`pm2go hub`): web dashboard, API, alert engine, Telegram notifier. One hub serves many hosts.

```
 host A: pm2go daemon + agent ──ws──┐
 host B: pm2go daemon + agent ──ws──┼──> hub (SQLite) ──> browser dashboard
 host C: pm2go daemon + agent ──ws──┘          └──> Telegram Bot API
```

Agents dial out to the hub (no inbound port on hosts). The hub never connects to hosts.

## Code map (`servers/pm2go`)

| Path | Role |
|------|------|
| `main.go`, `cli/` | commands, daemon spawn, systemd unit generation (`startup.go`), log tail |
| `common/` | `AppConfig`, `ProcessInfo`, `Event`, `LogLine`, RPC + agent protocol types, ecosystem loader, paths |
| `daemon/proc.go` | one supervised process: restart policy, soft restarts, graceful stop |
| `daemon/manager.go` | process table, instances, scale, rolling reload, monitor loop, save/resurrect |
| `daemon/logfile.go` | size-rotating log file with per-line hook |
| `daemon/watch.go` | polling file watcher for `watch` |
| `daemon/sampler_linux.go` | CPU/RSS per process group and host metrics from `/proc` |
| `daemon/sampler_other.go` | `ps`-based fallback so the code runs on a dev laptop; not for production |
| `daemon/server.go` | unix socket RPC (`~/.pm2go/rpc.sock`, mode 0600) |
| `agent/` | outbound WebSocket to the hub: snapshots, events, log batches, command execution |
| `hub/store.go` | SQLite schema (metrics, events, issues, alert rules and history, settings) |
| `hub/registry.go` | connected agents, latest snapshots, per-process log rings, crash tracking, live log fan-out |
| `hub/alerts.go` | rule model, evaluation state machine, escalation, reminders |
| `hub/telegram.go` | Telegram sender with per-chat minimum level, rate limit and retry |
| `hub/issues.go` | error-line fingerprinting and grouping |
| `hub/api.go`, `hub/auth.go` | HTTP API, session cookie auth, agent key check |
| `hub/ui/` | static dashboard (vanilla JS, no build step), embedded in the binary |

## Supervisor behaviour

- **Restart policy** matches PM2: a run shorter than `min_uptime` (default 1000 ms) is unstable. `max_restarts` (default 16) counts consecutive unstable runs, then status is `errored`. Stable runs reset the count. `exp_backoff_restart_delay` doubles per consecutive unstable run, capped at 15 s; otherwise `restart_delay`. `stop_exit_codes` end supervision.
- **Stop**: SIGTERM to the process group, SIGKILL after `kill_timeout` (default 1600 ms). Each child has its own process group.
- **Soft restarts** (`watch`, `cron_restart`, `max_memory_restart`) kill and restart at once, count in `restarts`, never count as unstable, and are reported with their own reason.
- **Instances**: `instances: N` runs N processes under one name (`PMGO_INSTANCE_ID` 0..N-1). There is no Node-style port sharing: apps that need one port use `SO_REUSEPORT` themselves. `scale` changes the count live. `reload` restarts one instance at a time and waits for each to stay up for `min_uptime`.
- **Logs**: the daemon pipes child stdout and stderr through a rotating writer (default 10 MB, keep 5 gzip files, `log_max_size` -1 turns rotation off). Every line also goes to the agent hook. **Consequence**: children are tied to the daemon. If the daemon dies, children get SIGPIPE on their next write. The systemd unit uses `KillMode=control-group` and `--resurrect` so a daemon restart brings everything back.
- **Metrics**: every 2 s the monitor reads `/proc` and sums CPU and RSS over each child's process group, plus host CPU, memory, load, disk (`/`), network (non-loopback) and uptime.
- **Start with an existing name** updates its config and restarts, so re-applying an ecosystem file is idempotent. Ecosystem files are JSON (`{"apps":[...]}`, `env_<profile>`, `--env-profile`, `--only`).
- **Environment**: the caller's environment is sent with each start request and never saved in the dump file. Children also get `PMGO_NAME`, `PMGO_ID`, `PMGO_INSTANCE_ID`, `PMGO_NAMESPACE`.

## Hub

- **Run**: `pm2go hub --listen :8080`. First run creates `~/.pm2go-hub/config.json` (override with `PMGO_HUB_HOME`) with a random admin password, agent key and session secret, and prints them once. `pm2go hub --show-keys` prints them again. `pm2go startup hub` installs a systemd unit.
- **Connect a host**: `pm2go link --hub http://hub:8080 --key <agent key> --name <machine>`. The same command is shown on the dashboard Settings page. `pm2go unlink` disconnects. Put TLS in front (reverse proxy) for anything beyond a trusted network; agents accept `https://`.
- **Auth**: dashboard login with the admin password sets a signed 7-day cookie (`SameSite=Strict`, `HttpOnly`); failed logins back off per IP. API scripts can send `Authorization: Bearer <admin password>`. Agents authenticate with the agent key. Browser WebSockets check the Origin.
- **Dashboard pages**: Servers overview with firing alerts, server detail (CPU, memory, load, disk, network charts, process table with restart / reload / stop / delete, recent events), process detail (per-process charts, config, live logs, scale, flush), Issues, Alerts (active, rules editor, history), Settings (link command, Telegram).
- **Data**: metrics stored every 10 s per host and per process in SQLite, bucketed down to about 240 points per chart. Events, alert history and metrics are pruned after `retention_days` (default 7). Live log history is an in-memory ring of 5000 lines per process, lost on hub restart.
- **Issues**: any log line matching `error|fatal|panic|exception|traceback|uncaught|unhandled|segfault` is fingerprinted (timestamps, numbers, hex ids and quoted values removed) and grouped with a count, first and last seen, and a status (open, resolved, ignored). A resolved issue reopens when it recurs.
- **Remote commands** allowed from the dashboard: restart, reload, stop, delete, flush, scale, start, save. The hub logs who (IP) ran what.

## Alerts and Telegram

- A **rule** has a level (`info`, `warning`, `critical`), a type, a scope (server and process, empty = all) and type-specific limits. Types: `system_cpu`, `system_memory`, `system_disk`, `system_load`, `proc_cpu`, `proc_memory`, `proc_status` (a process is `errored`), `restart_loop` (N crashes in a window), `log_errors` (N matching lines in a window, optional regex and stream), `server_offline`.
- **Sustained resource alerts**: a metric rule only fires after the condition holds continuously for `duration_s`. A brief spike resets the timer. With `repeat_s` the alert reminds you every N seconds while still firing ("still firing after 1h 5m").
- **Multiple levels**: create several rules of one type with rising thresholds, for example CPU > 80% for 5 min (warning) and CPU > 95% for 2 min (critical). While the higher level fires for the same target, the lower one stays quiet; if the higher clears first, the lower one then notifies.
- **Resolve**: the condition must be false for 10 s before the alert resolves. With `notify_resolved` and only if a firing message was sent, a "RESOLVED" message follows.
- **Defaults** are seeded on first run: process errored, restart loop (3 and 10 per 5 min), process CPU (80% / 95%), host memory (85% / 95%), host CPU, disk (85% / 95%), server offline (60 s), error burst (10 lines/min) and flood (100 lines/min). Edit or delete them in the dashboard.
- **Telegram**: create a bot with @BotFather, paste its token under Settings, add one or more chat ids (a user, group or channel the bot can write to), give each chat a minimum level (for example the on-call chat gets `critical` only, a team chat gets `warning` and up), press "Send test". A rule can also pin itself to specific chats. The token is stored in the hub database, shown only as "saved", and redacted from errors. Messages are rate limited to about one per second per queue and retried on failure or HTTP 429.

## Testing

`go test -race ./servers/pm2go/...`. Daemon tests use `sh`. Hub tests run the alert engine on a fake clock against a fake Telegram server and drive the HTTP API with a real WebSocket agent. Manual end-to-end check: start `pm2go hub`, `pm2go link` a daemon, start a noisy script and a crashing script, and watch the dashboard. Set `PMGO_HOME` to a short path (unix socket limit).

## Not built yet

The full checklist is in `pm2go-status.md`. Summary:

- Cluster-style port sharing and `wait_ready`, `ignore_watch` globs beyond name patterns, YAML ecosystem files, ssh `deploy`.
- Transaction tracing, CPU and heap profiling from the dashboard, deployment tracking (git revision per process), custom app metrics and actions SDK.
- Multi-user accounts, SSO, TLS in the hub itself, other alert channels (email, Slack, webhook).
- Log search across history (only the live ring is searchable by eye).
- Daemon restart without stopping children (children are tied to the daemon by the log pipes).
