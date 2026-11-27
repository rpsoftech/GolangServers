# mysql-backup-exe knowledge base

Reference for `servers/jwelly/mysql-backup-exe`: what it does, how it is built, why it is built that way, and what was learned running it. User-facing settings are in the tool's own `README.md`; this file keeps the reasoning.

## What it is

A Windows/Linux tool that backs up MySQL databases listed in `config.json` to gzipped SQL (`<name>-<yyyymmdd-hhmmss>.sql.gz`, `.age` when encrypted) and delivers them to one or more destinations. Two modes:

- **Quick run** (default): backs up every database once, prints a summary, waits for Enter when it owns a console. Exit code 1 if anything failed. Used by double-click, Task Scheduler and cron.
- **Daemon** (`--daemon`): runs each database on its own `cron` expression, optional `runOnStart`, desktop notifications on failure, self-update every 5 minutes.

## Code map

| File | Role |
|------|------|
| `main.go` | flags, quick run, daemon, summary output |
| `config.go` | config types, loading, `trimSpaces`, validation (all problems reported together) |
| `backup.go` | `RunBackup`: dump, verify, encrypt, deliver to every destination, retry, timeout |
| `dump.go` | engine selection (`auto`/`mysqldump`/`native`), `mysqldump` args, completeness check |
| `native.go` | pure Go dumper (no mysqldump needed), DSN builder |
| `definer.go` | optional `DEFINER` stripping |
| `dest.go`, `r2.go` | local folder, file server, Cloudflare R2 destinations and pruning |
| `encrypt.go` | age encryption, `keygen` and `decrypt` subcommands |
| `telegram.go` | Telegram reports |
| `selfupdate.go` | update check and restart for quick runs |
| `parallel.go` | worker pool (quick run) and gate (daemon) for `parallel` |
| `notify*.go` | desktop notifications (Windows and other) |

Shared updater lives in `utility/updater`; release automation in `utility/deploy` and `.github/workflows/release.yml`.

## Feature history (PRs on `rpsoftech/GolangServers`)

| PR | Change |
|----|--------|
| earlier | original tool: mysqldump and native engines, local + file server destinations, retention |
| #43 | fiber v3 migration (repo-wide) |
| #45 | auto-release changed components to STAGING on push to main |
| #48 | example configs for one-time and Linux cron use |
| #51 | optional per-database age encryption |
| #52 | Cloudflare R2 destination (S3 compatible, pruning) |
| #53 | version bump |
| #55 | `compress` setting, Telegram reports, update check and restart on quick runs (issue #54) |
| next | `parallel` setting, automatic `--column-statistics=0`, trimming spaces in config values |

## Design decisions

- **Dump is verified before it is trusted.** The stream must end with mysqldump's `Dump completed` line; a truncated dump is an error, never delivered.
- **Engine `auto`** tries `mysqldump`, then falls back to the native Go dumper. The fallback log shows only the first stderr line (`firstLine`), which can hide the real cause; look at the full error in the summary.
- **Password never on the command line.** It goes into a private temp option file passed as the first `--defaults-extra-file` argument (`mysqldump` requires that position), deleted afterwards.
- **`--column-statistics=0` only when supported.** MySQL 8 clients query a table old servers lack and exit with status 2. The tool runs `mysqldump --help` once per binary (cached) and adds the flag only if listed; older clients and MariaDB reject it.
- **`compress`** maps to `--compress` for mysqldump and DSN `compress=true` for the native engine (the driver's `Config.Compress` is unexported, so it goes through `cfg.Params`).
- **Encryption** happens once after verification; every destination of that database receives the same `.age` file. The secret key must live off the backup machine.
- **Telegram** sends one message per result, success or failure, from quick runs and the daemon. Failures to send are logged only, never fail the backup. The bot token is in the URL, so error text replaces it with `***`. Messages are clipped to fit the 4096-character limit.
- **Updates.** Daemon: `updater.RunDaemon` checks every 5 minutes and exits after installing (service manager restarts it). Quick run: checks once at start; if it installs a newer build it re-executes it with `--no-pause`, env `MYSQL_BACKUP_SKIP_UPDATE=1`, waits and returns its exit code, so the parent keeps the Enter pause. The daemon never re-executes itself. `--no-update` skips. Local (unpublished) builds never update. Releases are signed; the update channel must match the build (auto-release publishes STAGING only; PRODUCTION needs a manual release).
- **`parallel` (1 to 16, 0/unset = old behaviour).** Quick run: worker pool, results kept in config order. Daemon: a semaphore (`gate`) limits concurrent cron jobs and `runOnStart` jobs. Each running backup is one extra DB connection.
- **Restart path after an update.** The executable path is resolved BEFORE `SelfUpdate`. On Linux `os.Executable` follows the running file, which the update renames to `<exe>.old` and deletes, so resolving it afterwards gave `fork/exec .../mysql-backup-exe.old: no such file` (seen v7 to v8 on Linux).
- **Config whitespace.** Host, user, database, name, cron, paths and URLs are trimmed on load (a trailing space in a host gave `no such host`). Passwords and tokens are never trimmed.

## Performance lessons (remote vs local)

- Backing up from another machine is slow mainly because `mysqldump` asks about 8 to 10 questions per table, one round trip each. Measured case: MySQL 5.5.56, 172 tables per database, ping 9 ms, so about 0.12 s per table and about 28 s per database even with no data. Row volume was irrelevant.
- Ruled out by measurement: reverse DNS, `innodb_stats_on_metadata`, network bandwidth, compression (small dumps).
- Fixes, best first: run the tool on the MySQL server with host `127.0.0.1`; set `parallel` so databases overlap; `compress` only helps large dumps over slow links.
- A diagnosis lesson: guesses (DNS, stats) were wrong until timestamps from `mysqldump --verbose` were measured. Measure first.

## Testing and CI

- `go test ./servers/jwelly/mysql-backup-exe/` (unit tests use `httptest`, fake Telegram server, fake binaries; the column-statistics test uses a shell script and is skipped on Windows).
- CI runs against MySQL 5.5, 5.6, 5.7 and 8.0, a Windows build and test, and repo-wide `vet` and `test`.
- Run `-race` on concurrency changes. `gofmt -l` lists many files only because of `core.autocrlf` CRLF; harmless.

## Release process

Push to `main` auto-releases changed components to STAGING. PRODUCTION releases are manual and need explicit approval. Auto-update only reaches builds on the same channel.

## Working conventions for this repo

- No AI/tool attribution anywhere: commits, PRs, issues, branches, code comments (see `CLAUDE.md`).
- Branch names `feat/<topic>`, `fix/<topic>`, `docs/<topic>`.
- Squash-merge PRs. After a squash merge, branches show as "unmerged" in `git branch --no-merged`; check the PR state (`gh pr list --state all --head <branch>`) instead.
- Verify that a PR's last commits actually landed in `main` before calling it done (PR #55 was squash-merged before its final two commits were pushed, so they needed a follow-up PR).

## Open items

- Real-world check of Telegram, quick-run update restart, `compress` and `parallel` on production servers.
- PRODUCTION release of `mysql_backup_exe` once STAGING is verified.
- The fallback log could show the full error instead of the first line.
