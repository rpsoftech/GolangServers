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
| `hub/store.go`, `hub/migrations.go` | SQLite store; versioned schema migrations tracked in `schema_migrations` (metrics, events, issues, alert rules and history, settings) |
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
- **Instances**: `instances: N` runs N processes under one name (`PMGO_INSTANCE_ID` 0..N-1). `scale` changes the count live. `reload` restarts one instance at a time and waits for each to stay up for `min_uptime` (or to signal ready with `wait_ready`). To share one port between instances see **Shared socket and `wait_ready`**.
- **Logs**: the daemon pipes child stdout and stderr through a rotating writer (default 10 MB, keep 5 gzip files, `log_max_size` -1 turns rotation off). Every line also goes to the agent hook. **Consequence**: children are tied to the daemon. If the daemon dies, children get SIGPIPE on their next write. The systemd unit uses `KillMode=control-group` and `--resurrect` so a daemon restart brings everything back.
- **Metrics**: every 2 s the monitor reads `/proc` and sums CPU and RSS over each child's process group, plus host CPU, memory, load, disk (`/`), network (non-loopback) and uptime.
- **Start with an existing name** updates its config and restarts, so re-applying an ecosystem file is idempotent. Ecosystem files are JSON or YAML (`.json`, `.yaml`, `.yml`; an unknown extension is tried as JSON, then YAML) with one schema: `{"apps":[...]}`, `env_<profile>`, `--env-profile`, `--only`. YAML is parsed into generic data and re-marshalled to JSON, so both formats share one decoder (`common/ecosystem_yaml.go`); scalar `env` values such as `PORT: 3000` become strings. Unknown keys (for example `max_restart`) print `warning:` lines from the CLI and never fail the start; invalid YAML errors name the file and line.
- **`ignore_watch` patterns** (`daemon/glob.go`) are relative to `cwd`: `*` matches within one path segment, `**` matches across directories, a leading `/` anchors to `cwd`, a trailing `/` matches directories only, a pattern with an inner `/` (`a/b/c`) is anchored to `cwd`, and a bare name (`*.swp`, `cache`) matches at any depth. A match on a directory ignores everything below it. `.git`, `node_modules` and `.pm2go` are always ignored.
- **Environment**: the caller's environment is sent with each start request and never saved in the dump file. Children also get `PMGO_NAME`, `PMGO_ID`, `PMGO_INSTANCE_ID`, `PMGO_NAMESPACE`.

## Shared socket and `wait_ready`

Cluster-style port sharing without Node. Linux and macOS (unix platforms) only.

- **Config**: `listen` (`":8080"`, `"127.0.0.1:9000"` or `"unix:/run/app.sock"`), `listen_network` (`tcp` default, `tcp4`, `tcp6`, `unix`), `wait_ready` (bool), `listen_timeout` (ms, default 8000). CLI: `--listen`, `--wait-ready`, `--listen-timeout`. Without them nothing changes.
- **Socket**: the daemon opens the listener once per app name (a bind error fails `start`) and passes it to every instance as inherited fd 3 (systemd socket activation convention): `LISTEN_FDS=1`, `LISTEN_FDNAMES=<name>`, plus `PMGO_LISTEN=<addr>`. `LISTEN_PID` is not set (the pid is unknown before the fork); `sdk/runtime` ignores it when absent. The daemon closes the socket on `delete` and when `scale` removes the last instance; a stopped app keeps it, so connections queue until the next start. Changing `listen` and re-running `start` swaps the socket; running children keep the old one until they restart.
- **Readiness**: with `wait_ready` the child gets a pipe write end as fd `PMGO_READY_FD` (3 without `listen`, 4 with it) and must write `ready\n` once. The status is `launching` until then. After `listen_timeout` the child is stopped (SIGTERM, then SIGKILL after `kill_timeout`), the run counts as unstable (also when it exits before signalling ready, however long it ran) and the normal restart policy applies. Event reason `ready_timeout`.
- **Reload**: with `listen` the new instance starts next to the old one, `reload` waits for ready (or `min_uptime` without `wait_ready`), then sends SIGTERM to the old one and waits up to `kill_timeout` before the next instance. If the new instance does not become ready, the old one keeps running and `reload` returns an error. Without `listen` the order stays stop, start, wait (for ready when `wait_ready`).
- **Apps must drain**: on SIGTERM stop accepting, finish in-flight requests, exit. Because the daemon holds the socket, queued connections wait for the next instance instead of being refused. A connection accepted at the instant of the signal can still be lost if the server exits before it registers it, so close the listener, wait a short moment and then call `Shutdown` (see the example).
- **Helper** `github.com/rpsoftech/golang-servers/servers/pm2go/sdk/runtime` (standard library only): `Listener()` (inherited listener, else `:$PORT` or `:8080`), `ListenerOn(network, addr)`, `Inherited()`, `Ready()`. Outside pm2go they fall back to a normal listener and `Ready` does nothing.

```go
ln, err := runtime.ListenerOn("tcp", ":8080") // shared socket under pm2go
if err != nil { log.Fatal(err) }
srv := &http.Server{Handler: mux}
go srv.Serve(ln)
runtime.Ready() // after init: reload and start wait for this
sig := make(chan os.Signal, 1)
signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
<-sig
ln.Close()                          // stop accepting
time.Sleep(200 * time.Millisecond)  // let a just-accepted connection register
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
srv.Shutdown(ctx)                   // drain in-flight requests
```

Start it with `pm2go start ./app --name web --instances 3 --listen :8080 --wait-ready`.

## Hub

- **Run**: `pm2go hub --listen :8080`. First run creates `~/.pm2go-hub/config.json` (override with `PMGO_HUB_HOME`) with a random admin password, agent key and session secret, and prints them once. `pm2go hub --show-keys` prints them again. `pm2go startup hub` installs a systemd unit.
- **Connect a host**: `pm2go link --hub http://hub:8080 --key <agent key> --name <machine>`. The same command is shown on the dashboard Settings page. `pm2go unlink` disconnects. Use TLS for anything beyond a trusted network: a reverse proxy, or the hub's own `--tls-cert/--tls-key` or `--acme-domain` (see `pm2go-deploy.md`). Agents accept `https://`; `--hub-ca FILE` trusts a private CA.
- **Auth**: dashboard login with the admin password sets a signed 7-day cookie (`SameSite=Strict`, `HttpOnly`); failed logins back off per IP. API scripts can send `Authorization: Bearer <admin password>`. Agents authenticate with the agent key. Browser WebSockets check the Origin.
- **Dashboard pages**: Servers overview with firing alerts, server detail (CPU, memory, load, disk, network charts, process table with restart / reload / stop / delete, recent events), process detail (per-process charts, config, live logs, scale, flush), Issues, Alerts (active, rules editor, history), Settings (link command, Telegram).
- **Schema**: the hub database is versioned. On start the hub applies pending migrations in order, each in one transaction. A database from before versioning is adopted as version 1. A database with a newer version than the binary knows is refused with an error (no downgrade).
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

- ssh `deploy`.
- Transaction tracing, CPU and heap profiling from the dashboard, deployment tracking (git revision per process), custom app metrics and actions SDK.
- Multi-user accounts, SSO, other alert channels (email, Slack, webhook).
- Log search across history (only the live ring is searchable by eye).
- Daemon restart without stopping children (children are tied to the daemon by the log pipes).
