# pm2go developer guide

Read this instead of re-reading the whole code. It is the map: what each package owns, the types that cross package borders, how to add the common kinds of change, and the rules every change follows. Behaviour and design reasons are in `pm2go.md`, deployment in `pm2go-deploy.md`, what is done or open in `pm2go-status.md`, and who is working on what in `pm2go-work-log.md`.

## Process model

```
CLI ──unix socket JSON-RPC──> daemon (Manager, Procs) ──hooks──> agent ──WebSocket──> hub ──> browser
                                   │                                                   └──> Telegram
                                   └── child processes (own process group, stdout/stderr piped through RotatingFile)
```

- One Go module (`github.com/rpsoftech/golang-servers`), one binary `servers/pm2go`, three roles chosen by subcommand: CLI, `daemon`, `hub`.
- Linux is the production target. `daemon/sampler_other.go` is a `ps` fallback only so code runs on a Mac during development.
- Go 1.26, standard library `net/http` (not Fiber) for the hub, `gorilla/websocket`, `modernc.org/sqlite` (pure Go, `CGO_ENABLED=0` works), `robfig/cron/v3`.

## Package map and what lives where

| Package | Files | Owns |
|---------|-------|------|
| `common` | `types.go` | `AppConfig` (the per-app config, also the ecosystem and dump format), `ProcessInfo`, `Event`, `LogLine`, status constants, default getters (`MaxRestartsOrDefault`, ...) |
| | `rpc.go` | CLI-to-daemon method names and params (`Method*`, `StartParams`, `TargetParams`, `ScaleParams`) |
| | `proto.go` | agent-to-hub protocol (`Msg*` names, `Envelope`, `Hello`, `Snapshot`, `Command`, `CommandResult`), `SystemSample`, `LinkConfig` and `link.json` helpers |
| | `ecosystem.go` | ecosystem file loader (`LoadEcosystem`), `DefaultName` |
| | `paths.go` | `Home()` (`$PMGO_HOME` or `~/.pm2go`), file path helpers, `SafeName` |
| `daemon` | `proc.go` | `Proc`: one supervised process, the `loop` goroutine, restart policy, soft restarts (`requestRestart`), `Stop`, log writers, `info()` |
| | `manager.go` | `Manager`: process table, `Start` (idempotent by name), `Stop`, `Restart`, `Reload`, `Scale`, `Delete`, `Save`, `Resurrect`, `Flush`, `RunMonitor` (sampling and memory limit), `SetHooks` |
| | `server.go` | unix socket server, `Run(Options)`, RPC `handle` switch |
| | `logfile.go` | `RotatingFile` (size rotation, gzip, per-line callback, `Truncate`) |
| | `watch.go` | polling file watcher used by `watch` |
| | `sampler_linux.go` | `/proc` sampling: `Sampler.SampleProcs`, `SampleSystem` |
| | `sys_unix.go`, `shell.go` | process group attrs, signal helpers, `detachAttr` |
| `agent` | `agent.go` | `Agent`: reads `link.json`, dials the hub, sends snapshots (2 s), events, log batches (500 ms), executes hub commands (`exec`) |
| `cli` | `commands.go` | subcommand dispatch (`Main`), flag parsing (`parseInterspersed`), `cmdStart` and friends |
| | `client.go` | `call()` RPC client, daemon auto-spawn |
| | `startup.go` | systemd unit rendering, `Version` variable (set with `-ldflags -X`) |
| | `format.go`, `logs.go` | table output, `logs` tail |
| `hub` | `server.go` | `Run`: wires store, registry, engine, notifier, HTTP server, background tickers |
| | `config.go` | `config.json` (admin password, agent key, session secret), `Home()` (`$PMGO_HUB_HOME`) |
| | `store.go` | `OpenStore` (runs migrations), settings key/value, `Prune` |
| | `migrations.go` | `migrations` list (versioned SQL or Go-func steps), `migrate`, `schema_migrations` table |
| | `registry.go` | connected agents, latest `ServerState`, log rings, crash tracking, live-log subscribers, `Command` (send command, wait result) |
| | `alerts.go` | `Rule`, `Engine` (state machine, escalation, reminders), rule CRUD, history |
| | `telegram.go` | `Notifier`, `TelegramSettings`, `FormatAlert` |
| | `issues.go` | error-line classification and fingerprinting |
| | `api.go`, `auth.go` | routes, handlers, session cookie, agent key, `trustProxy` |
| | `ui/` | `index.html`, `style.css`, `app.js` embedded with `go:embed ui` |
| `deploy` | Dockerfile, compose, Caddy files | see `pm2go-deploy.md` |

## Key flows

- **Start**: CLI `cmdStart` builds `AppConfig` (or loads ecosystem) -> RPC `start` -> `Manager.Start` creates or updates `Proc`s by name -> `Proc.Start` spawns `loop`.
- **Supervise**: `loop` starts the child with stdout/stderr bound to two `RotatingFile`s, waits on exit, a stop request, or a soft restart request (`watch`, `cron`, `max_memory`). Exit handling decides restart, `errored` or `stopped`, and emits `common.Event`s through the manager hook.
- **Telemetry**: `Manager.RunMonitor` samples every 2 s and caches usage in each `Proc`. The agent builds a `Snapshot` from `Manager.List()` plus `Manager.System()`.
- **Hub ingest**: `API.agentWS` reads envelopes and calls `Registry.OnSnapshot / OnEvent / OnLogs`. Snapshots are persisted every 10 s. Events go to `events`. Log lines go to a per-process ring, the issue grouper and live subscribers.
- **Alerts**: `Engine.Evaluate` runs every 2 s over `Registry.States()`, builds observations per rule, drives each (rule, server, process) state machine (pending, firing, resolved), applies level suppression, then calls `Notifier.Notify`.
- **Remote command**: browser POST `/api/servers/{name}/command` -> `Registry.Command` -> agent `exec` -> `Manager.*` -> `CommandResult` back.

## How to add common things

| Change | Steps |
|--------|-------|
| New CLI command | add a `case` in `cli.Main`, write `cmdX` in `cli/commands.go` (or a new file), update the `usage` text |
| New daemon RPC | constant in `common/rpc.go`, params struct there, case in `daemon/server.go` `handle`, a `Manager` method |
| New `AppConfig` field | add to `common/types.go` with a json tag and a default getter, use it in `daemon`, add a `start` flag in `cmdStart`, document in `pm2go.md`; keep old dump files loading (new fields must default sensibly) |
| New agent -> hub message | `Msg*` const and payload struct in `common/proto.go`, producer in `agent/agent.go` (`a.send(...)`), switch case in `hub/api.go` `agentWS`, handler in `hub/registry.go` |
| New hub -> agent command | action string handled in `agent.exec`, add to `allowedActions` in `hub/api.go` |
| New hub table or schema change | add a migration: append `{version: N+1, name: "...", sql: ...}` (and/or `fn: func(tx *sql.Tx) error` for data fixes) to `migrations` in `hub/migrations.go`. Never edit a released migration. Each runs in one transaction with its `schema_migrations` row; a failure aborts startup. A database newer than the binary is refused. Test it in `hub/store_test.go` |
| New alert rule type | const + `Validate` + `observe` in `hub/alerts.go`, `RULE_TYPES` and `describeRule` in `ui/app.js`, a test in `hub/hub_test.go` using the fake clock |
| New notification channel | implement it beside `Notifier` in `hub/telegram.go` or a new file, route `Engine` sends through a small channel interface (do this refactor once, in the channels issue) |
| New API route | register in `API.Handler` with `req(...)` for auth, handler in `hub/api.go` (or a new file in package `hub`) |
| New dashboard page | add a `page*` function and a `case` in `route()` in `ui/app.js`, add a nav entry in `NAV` |

## Rules every change follows

- **No AI tooling mentions** anywhere (commits, PRs, issues, code comments, branch names). Commits are authored by the repository owner only, no co-author trailers. Branch names are plain: `feat/<topic>`, `fix/<topic>`.
- Conventional commit titles: `feat(pm2go): ...`, `fix(pm2go): ...`, `docs(pm2go): ...`.
- UI: build DOM with the `h()` helper and `textContent`. Never use `innerHTML` with data from logs, process names or API fields. The CSP forbids inline scripts.
- Secrets never leave the hub through the API (the Telegram token is shown only as `token_set`). Keep it that way for any new secret.
- Every hub or daemon behaviour change gets a test. Alert logic is tested with the fake clock in `hub/hub_test.go` (`setup`, `setSnapshot`, `advance`).
- Update `docs/knowledge-base/pm2go.md` (behaviour), `pm2go-status.md` (tick boxes), and add your entry to `pm2go-work-log.md` in the same PR.
- Keep the Linux production build green.

## Commands you will need

```bash
gofmt -l servers/pm2go                                   # must print nothing
go vet ./servers/pm2go/...
go test -race -count=1 ./servers/pm2go/...               # ~20 s
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dev/null ./servers/pm2go
```

Manual end-to-end run on a laptop (use a short `PMGO_HOME`, unix socket paths are limited to about 104 bytes on macOS):

```bash
go build -o /tmp/pm2go ./servers/pm2go
export PMGO_HUB_HOME=$(mktemp -d /tmp/hub.XXXX) PMGO_HOME=$(mktemp -d /tmp/pg.XXXX)
/tmp/pm2go hub --listen 127.0.0.1:18080 &            # prints admin password and agent key
/tmp/pm2go start ./some-script.sh --name demo
/tmp/pm2go link --hub http://127.0.0.1:18080 --key <agent key> --name dev-box
# open http://127.0.0.1:18080 and sign in
/tmp/pm2go kill; kill %1                              # cleanup, check "pgrep -fl pm2go" is empty
```

## Hot spots (files many changes touch)

Merge conflicts happen here, so keep edits small and in separate functions or new files where you can:

- `hub/ui/app.js` (all pages), `hub/api.go` (all routes), `hub/migrations.go` (append-only list), `common/types.go`, `common/proto.go`, `agent/agent.go`, `cli/commands.go`.
- Prefer adding a new file (`hub/xyz.go`, `ui/xyz.js` plus a `<script>` tag) and a one-line registration over editing the big files in many places.
- Rebase on the integration branch before opening your PR and re-run the whole test suite.

## Known gaps in the base code (fix when your issue touches them)

- `cli` has no tests.
- The hub UI has no automated tests (checked by hand in a browser).
- Hub restart loses the live log ring.
- `Registry.OnLogs` holds the registry lock while fanning out to subscribers (non-blocking sends, but still a lock on the hot path).
