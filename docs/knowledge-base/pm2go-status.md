# pm2go status checklist

Last updated with the Docker and Caddy deployment files. Tick boxes as you verify. Deployment steps are in `pm2go-deploy.md`, design in `pm2go.md`.

## Built (in the repository, covered by tests unless noted)

### Process manager (daemon and CLI)
- [x] start / stop / restart / delete / list / describe by id, name, namespace or `all`
- [x] restart policy: autorestart, `max_restarts`, `min_uptime`, `restart_delay`, exponential backoff, `stop_exit_codes`
- [x] graceful stop (SIGTERM to process group, then SIGKILL after `kill_timeout`)
- [x] instances, `scale`, rolling `reload`
- [x] `watch` (polling), `cron_restart`, `max_memory_restart`
- [x] log files with size rotation and gzip, `logs` tail, `flush`
- [x] `save` / `resurrect`, JSON ecosystem files with `env_<profile>`, `--only`
- [x] `startup` / `unstartup` systemd unit generation (generation is unit-tested by build only; install not run on a real host)
- [x] CPU, memory, host metrics from `/proc` (written for Linux, cross-compiled, **not run on Linux yet**)

### Hub and dashboard
- [x] agent connection (outbound WebSocket), snapshots, events, log batches, remote commands
- [x] SQLite storage, retention pruning
- [x] dashboard: servers, server detail with charts, process detail with live logs, issues, alerts, settings
- [x] login with session cookie, agent key, login back-off, WebSocket origin check
- [x] issues grouping from error log lines

### Alerts and Telegram
- [x] leveled rules (info, warning, critical) with sustained duration, reminders, resolved messages, escalation
- [x] rule types: host CPU, memory, disk, load, process CPU, memory, errored, restart loop, log error rate, server offline
- [x] 13 default rules seeded
- [x] Telegram sender with per-chat minimum level, rate limit, retry (tested against a fake server, **not against the real Telegram API**)

### Deployment files
- [x] `Dockerfile`, `docker-compose.yml`, `Caddyfile.host`, `Caddyfile.container`, `.env.example`
- [x] reverse proxy support (`PMGO_TRUST_PROXY`), `/healthz`
- [ ] **not built or run**: Docker is not installed on the development machine, so the image and compose stack are untested

## Installed / required on your platform (check each)

- [ ] Docker and compose plugin on the hub server
- [ ] DNS record for the dashboard name pointing at the hub server
- [ ] Caddy on the platform, with ports 80 and 443 reachable (option A) or the compose Caddy profile (option B)
- [ ] Telegram bot created with @BotFather, bot token saved
- [ ] chat ids known and the bot added to each group or started by each user
- [ ] `pm2go` binary on each monitored host (`/usr/local/bin/pm2go`)
- [ ] systemd on each monitored host (for `pm2go startup`)
- [ ] outbound HTTPS from each host to the hub

## Pending work

### Verify first (nothing here is proven on real infrastructure)
- [ ] build the Docker image and run the compose stack
- [ ] reload host Caddy with the new block and open the dashboard over HTTPS
- [ ] run the daemon on a real Linux host and check `/proc` CPU and memory numbers against `top`
- [ ] `sudo pm2go startup daemon`, reboot the host, confirm processes come back
- [ ] real Telegram test message and one full alert cycle (firing, reminder, resolved)
- [ ] load test: many processes and chatty logs, watch hub CPU and database size

### Features not built yet
- [ ] SDK for custom app metrics and custom actions (PM2 Plus custom metrics / actions)
- [ ] transaction tracing (HTTP request traces)
- [ ] CPU and heap profiling from the dashboard (pprof)
- [ ] deployment tracking (git revision per process, deploy markers on charts)
- [ ] `pm2go deploy` over ssh
- [ ] daemon restart without stopping children (today children are tied to the daemon by log pipes)
- [ ] cluster-style port sharing and `wait_ready` readiness
- [ ] YAML ecosystem files, richer `ignore_watch` globs
- [ ] more alert channels (email, Slack, webhook), quiet hours, alert silencing
- [ ] multi-user accounts, roles, SSO, audit log
- [ ] TLS inside the hub (today Caddy terminates TLS)
- [ ] long-range log storage and search (today: 5000-line in-memory ring per process)
- [ ] `monit` terminal dashboard
- [ ] GitHub Actions workflow to build and publish the image and Linux binaries

### Known limits
- Hub restart loses the in-memory live log history (metrics, events, issues, alerts persist).
- One admin password for the whole dashboard.
- `startup` installs a system-level unit; user-level systemd units are not generated.
- The `ps`-based fallback sampler is for development on macOS only.
