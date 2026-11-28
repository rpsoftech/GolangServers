# PM2 and PM2 Plus: feature analysis for a Go rewrite

Status: analysis only. No code yet. Sources: PM2 docs (pm2.keymetrics.io/docs), PM2 Plus docs (doc.pm2.io, github.com/keymetrics/doc-pm2), pm2.io/pricing, plus prior knowledge of the PM2 codebase. Tier details for PM2 Plus change often; verify at app.pm2.io before copying any limit.

## 1. PM2 Runtime (open source, the CLI + daemon)

### 1.1 Architecture
- `pm2` CLI talks to a long-lived **God daemon** over a local pub/sub + RPC socket (Unix socket on Linux/macOS, named pipe on Windows). First CLI call spawns the daemon.
- God daemon forks and supervises child processes, owns the process table, and publishes events (`process:event`, `log:out`, `log:err`).
- Dump file (`~/.pm2/dump.pm2`) stores the saved process list for resurrect.
- State dir `~/.pm2/`: `logs/`, `pids/`, `modules/`, `pm2.log`, `pm2.pid`, `rpc.sock`, `pub.sock`.

### 1.2 Process lifecycle
- start / stop / restart / reload / delete / reset / kill daemon.
- Auto-restart on crash, with `restart_delay`, `exp_backoff_restart_delay`, `max_restarts`, `min_uptime`, `autorestart`.
- Statuses: `online`, `stopping`, `stopped`, `launching`, `errored`, `one-launch-status`, `waiting restart`.
- Graceful shutdown: SIGINT, then `kill_timeout`, then SIGKILL. Options `shutdown_with_message`, `kill_retry_time`.
- `wait_ready` + `listen_timeout`: app signals readiness (`process.send('ready')`). Go equivalent needs a readiness protocol (env var + unix socket or stdout token).
- `stop_exit_codes`: exit codes that do not trigger restart.
- Cron restart (`cron_restart`), watch mode (`watch`, `ignore_watch`, `watch_delay`, `watch_options`).
- Memory limit restart (`max_memory_restart`).
- Fork mode vs cluster mode.

### 1.3 Cluster mode (Node-specific)
- `-i <n|max>` uses Node `cluster` to share one port among instances. Round-robin balancing.
- `reload` = zero-downtime rolling restart.
- `scale <app> <+n|n>` adds or removes instances live.
- **Go equivalent:** Go has no cluster module. Options: (a) SO_REUSEPORT instances, (b) the manager holds the listening socket and passes it as fd 3 (systemd socket activation style, `LISTEN_FDS`), (c) built-in TCP/HTTP reverse proxy. Recommend (b) first, (c) later. Zero-downtime reload needs start new, wait ready, stop old.

### 1.4 CLI command surface
- Lifecycle: `start`, `stop`, `restart`, `reload`, `delete`, `reset`, `scale`, `sendSignal`, `kill`, `ping`.
- Inspect: `list`/`ls`/`status`, `jlist`, `prettylist`, `describe`/`show`, `monit`, `logs`, `flush`, `reloadLogs`, `env`, `id`, `pid`.
- Persistence: `save`, `resurrect`, `startup [platform]`, `unstartup`.
- Config: `ecosystem`/`init` (generate sample), `start ecosystem.config.js --only --env`, `set`, `get`, `conf`, `config`, `unset`.
- Modules: `install`, `uninstall`, `publish`, `module:generate`, `module:update`.
- Deploy: `deploy <file> setup|update|revert|curr|prev|exec|list`.
- Misc: `update`, `report`, `serve`/`expose` (static file server), `link`/`unlink` (PM2 Plus), `open`, `plus`, `trigger <app> <action>`, `profile:cpu|mem`, `attach`, `inspect`, `gracefulReload`, `dump`, `launchd`.
- Flags: `--name`, `--watch`, `--max-memory-restart`, `--instances`, `--cron`, `--interpreter`, `--interpreter-args`, `--node-args`, `--log`, `--output`, `--error`, `--merge-logs`, `--time`, `--env`, `--update-env`, `--no-autorestart`, `--no-daemon`, `--no-vizion`, `--no-treekill`, `--silent`, `--namespace`, `-f`, `--json`, `--lines`, `--raw`, `--nostream`, `--timestamp`, `--only`, `--cwd`, `--uid`, `--gid`.

### 1.5 Ecosystem / declaration file
- Formats: JS, JSON, YAML. Go rewrite should do JSON + YAML + TOML (no JS evaluation).
- `apps[]` fields: `name`, `script`, `cwd`, `args`, `interpreter`, `interpreter_args`, `exec_mode`, `instances`, `watch`, `ignore_watch`, `max_memory_restart`, `env`, `env_<name>` (merged with `--env`), `source_map_support`, `instance_var`, `filter_env`, `treekill`, `namespace`, `node_args`, `time`, `out_file`, `error_file`, `log_file`, `log_date_format`, `merge_logs`, `pid_file`, `uid`, `gid`, `autorestart`, `cron_restart`, `vizion`, `post_update`, `force`, `kill_timeout`, `listen_timeout`, `wait_ready`, `increment_var`, `restart_delay`, `exp_backoff_restart_delay`, `max_restarts`, `min_uptime`, `stop_exit_codes`.
- `deploy` block per environment: `user`, `host` (list), `ref`, `repo`, `path`, `ssh_options`, `pre-setup`, `post-setup`, `pre-deploy-local`, `post-deploy`, `pre-deploy`, `env`.

### 1.6 Logs
- Per-process out/err files under `~/.pm2/logs/<name>-out.log`, `-error.log`, `-<id>.log` naming.
- Live `pm2 logs` tail of all or one process, `--lines`, `--json`, `--timestamp`, `--raw`.
- `flush`, `reloadLogs` (reopen after external rotation).
- **pm2-logrotate module**: `max_size`, `retain`, `compress`, `dateFormat`, `rotateModule`, `workerInterval`, `rotateInterval` (cron).
- Go: built-in rotation instead of a separate module.

### 1.7 Monitoring (local)
- `pm2 monit`: terminal dashboard with process list, per-process CPU/memory, live logs, metadata. TUI via Bubble Tea in Go.
- Metrics sampled via `pidusage` (CPU %, memory RSS). Go: `gopsutil`.
- Process metadata via `vizion`: git revision, branch, remote, comment, update time.
- `describe` shows: status, name, id, pid, restarts, unstable restarts, uptime, script path, args, interpreter, exec cwd, error/out log path, mode, watching, created at, git info, custom metrics, divergent env vars.

### 1.8 Startup scripts
- `pm2 startup` detects init system: systemd, upstart, launchd, openrc, rcd, systemv, smf. Prints or runs a sudo command that installs a service running `pm2 resurrect`.
- `pm2 save` writes dump; `pm2 unstartup` removes service.
- Go: generate unit files for systemd + launchd first; others later.

### 1.9 Module system
- Modules are npm packages run as managed processes under the daemon (logrotate, pm2-server-monit, pm2-auto-pull, etc.).
- Go replacement: built-in features first, plus a plugin protocol (child process speaking JSON over a socket) later.

### 1.10 Programmatic API
- `pm2.connect/start/stop/restart/reload/delete/list/describe/launchBus/sendDataToProcessId/...` from Node.
- Go: a Go client package + the same JSON-RPC over a unix socket, so any language can drive it.

### 1.11 Application-side integration (PMX / @pm2/io)
- Metrics: `metric`, `counter`, `meter`, `histogram`. Actions: `action('name', cb)`. Notifications: `notify(err)`. Probes: event-loop lag, heap, GC, HTTP latency, active handles.
- Go: SDK package `pm2go/sdk` exposing metrics, actions, error reporting, over a unix socket back to the daemon. Auto-collected runtime metrics from `runtime.MemStats`, goroutine count, GC pauses.

### 1.12 Others
- Namespaces for grouping apps.
- Environment variable injection: `PM2_HOME`, `NODE_APP_INSTANCE`, `pm_id`, `pm_cwd`, etc. Go: `PMGO_HOME`, `PMGO_INSTANCE_ID`.
- Docker mode (`pm2-runtime`): foreground, PID 1 friendly, signal forwarding.
- Windows support, `uid`/`gid` drop on Unix.
- Static file server: `pm2 serve <dir> <port> --spa`.
- Deploy tool: SSH, git-based, release dirs, `current` symlink, hooks, revert.

## 2. PM2 Plus (hosted SaaS dashboard, app.pm2.io)

Linked via `pm2 link <secret> <public> [machine-name]`. A **KM agent** (`@pm2/agent`) inside the daemon opens an outbound WebSocket to the Plus cloud (no inbound port), streams metrics, logs, exceptions, and receives remote commands.

### 2.1 Core (all paid plans)
1. **Organization / bucket model**: bucket = one monitored project (a set of servers). Users and roles per bucket. Bucket secret/public key pair.
2. **Servers view**: every linked machine, online/offline, OS, CPU count, memory, uptime, agent version, IP.
3. **Real-time dashboard**: per-server and per-process CPU, memory, event loop, heap, HTTP req/min, latency, restarts, uptime. Live charts, history picker.
4. **Process list/management in browser**: restart, reload, stop, delete, scale from the web UI (remote actions through the agent).
5. **Logs**: live streaming of stdout/stderr per app/server, searchable, filter by process/server; stored logs (retention is a higher-tier feature).
6. **Issues dashboard (exceptions)**: uncaught exceptions and `notify` errors grouped by fingerprint, count, first/last seen, stack trace, affected process/server, status (new, resolved, ignored).
7. **Custom metrics**: any app-exposed metric displayed as a chart. Metric types: gauge, counter, meter, histogram.
8. **Custom actions**: buttons in the UI that call app-registered functions (with optional params), result shown.
9. **Notifications / alerting**: rules on metrics (CPU > x, memory > y, process down, restart loop, exceptions), channels: email, Slack, webhook. Conditions: threshold, duration, severity.
10. **Server monitoring (system)**: CPU, RAM, disk, network I/O, load average, process count, via `pm2-server-monit`. Also Docker container and DB monitors (Mongo, Redis, Postgres, MySQL...) as plugins.
11. **Data retention**: ~ 1 month standard up to ~1 year on top tier; processes over plan limit ignored.

### 2.2 Advanced (higher tier / Enterprise)
12. **Transaction tracing**: HTTP request traces with spans (DB, outbound HTTP, Redis), per-route latency percentiles, slow-request list, trace waterfall.
13. **In-situ profiling**: remote CPU profile and memory (heap) snapshot triggered from the UI, downloaded/visualized (flamegraph).
14. **Deployment tracking**: git revision/branch/comment from `vizion`, deploy events marked on charts, diff between revisions.
15. **Dashboards customization**: user-defined boards with widgets.
16. **Advanced alerting rules**.
17. **Log storage / search over long ranges**.
18. **Team features**: multiple members, invites, permissions, bucket sharing, audit.
19. **API / integrations**: REST API and webhooks; Slack, email, PagerDuty-style.
20. **Auth**: SSO / OAuth login (GitHub, Google), API tokens.

Not confirmed from public pages: remote heap snapshot specifics, deployment tracking specifics. Marked as best knowledge, to verify by hands-on trial of a Plus trial account if wanted.

## 3. Proposed Go design (working name `pm2go`, single static binary)

Components, each its own Go package under `servers/pm2go/` (CLI + daemon) and `servers/pm2go-cloud/` (dashboard):

| Component | Role | Notes |
|---|---|---|
| `pm2go` CLI | cobra commands matching PM2 names | same verbs so PM2 users switch easily |
| Daemon | process supervisor, log writer, metrics sampler, event bus | unix socket JSON-RPC, started on demand |
| Supervisor | fork/exec, restart policy, graceful stop, process groups | `os/exec`, `syscall.SysProcAttr{Setpgid}` |
| Config | ecosystem JSON/YAML/TOML parser, env profiles | validator from repo `validator/` |
| Logs | file writer, rotation, tail, fan-out to subscribers | built-in rotation |
| Metrics | gopsutil CPU/mem/IO per pid, ring buffer | feeds `monit` and agent |
| TUI | `monit` | Bubble Tea |
| Startup | systemd/launchd unit generation | |
| Deploy | SSH + git release dirs, hooks, revert | `x/crypto/ssh` |
| SDK | in-app metrics/actions/errors | Go package, optional |
| Agent | outbound WebSocket to cloud, streams, receives commands | like KM agent |
| Cloud server | ingest API, auth, buckets, alert engine, log store, REST + WS | Fiber (matches repo servers), MongoDB or MySQL from `utility/`, Redis for live fan-out |
| Web UI | dashboard SPA | served by cloud server |

Existing repo helpers to reuse: `utility/mongodb`, `utility/mysql`, `utility/redis`, `utility/jwt`, `middleware/`, `env/`, `utility/updater`, `utility/deploy`.

### Phases
1. **MVP supervisor**: daemon + CLI start/stop/restart/delete/list/describe/logs, restart policy, save/resurrect, ecosystem file.
2. **Runtime parity**: watch, cron restart, max-memory restart, log rotation, `monit` TUI, startup scripts, namespaces, reload with readiness, socket-passing scale.
3. **SDK + agent**: metrics, actions, error capture; `link` command; agent protocol.
4. **Cloud MVP**: buckets, servers, live dashboard, logs, remote process control, issues, alerts (email/Slack/webhook).
5. **Advanced**: transaction tracing (HTTP middleware SDK, OpenTelemetry-compatible), pprof CPU/heap remote capture, deployment tracking, deploy command, custom dashboards, retention tiers, SSO.

### Key design risks
- Node cluster semantics do not map to Go; use fd passing or built-in proxy.
- Windows: process groups and signals differ; scope Windows later.
- Agent security: outbound-only WS, per-bucket key, TLS, command allowlist, signed commands.
- Zero-downtime reload requires a readiness contract for non-Go apps (any script/binary can be managed, not only Go).

## 4. Decisions
- Internal use only (decided): no billing, plans, or retention tiers. Single tenant first.
- Phase 1 supervisor MVP built; see `pm2go.md`.

## 5. Open decisions
- Manage only Go binaries or any command (PM2 manages any interpreter)? Recommend any command.
- Cloud storage choice: MongoDB (events, logs) vs MySQL; time-series for metrics (ring in Redis + downsampled in DB).
- Build self-hosted dashboard only, or multi-tenant SaaS from start. Recommend self-hosted single-tenant first, multi-tenant buckets in schema from day one.
