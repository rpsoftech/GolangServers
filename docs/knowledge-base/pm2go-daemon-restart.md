# pm2go: daemon restart keeps managed processes running

Status: implemented (issue #99).

## Problem

The daemon used to pipe each child's stdout and stderr through its own writers. When the daemon died, children got SIGPIPE on their next write, and the systemd unit used `KillMode=control-group`, so a daemon upgrade, crash or `systemctl restart` restarted every application.

## Design

### 1. Logs go to files, the daemon tails them

- The child gets the log files themselves as stdout and stderr (`*os.File`, opened `O_APPEND`), so there is no pipe and no copy goroutine. Writes survive a daemon death.
- One `logTail` per file reads new bytes by offset every 200 ms (polling, portable; inotify is not needed at this volume), splits lines (a partial last line waits for its newline) and calls the per-line hook for the agent.
- A tailer starts at the end of the file when it is created (the old writer also only reported new lines), so a new daemon does not replay history.
- Truncation (`flush`, rotation) is detected when the size is smaller than the offset; the offset resets to 0.
- The supervisor drains the tailer when a child exits, so the last lines are reported before the `exit` event.

### 2. Rotation by copy-truncate

When the file is larger than `log_max_size` the tailer first reads to the end, copies the file to `<name>.<ts>.rot`, truncates it, and gzips the copy to `name.1.gz` in the background (older files shift to `.2.gz` ... `.N.gz`, as before). `O_APPEND` makes the child continue at the new end. **Trade-off**: bytes written between the copy and the truncate are lost (a few lines under a very chatty app). Rename plus signal was rejected: it needs the app to reopen its files.

### 3. State file

`$PMGO_HOME/state.json` (mode 0600; it holds each app's environment, which the dump file does not, because a respawn after adoption needs it) is written atomically (temp file, fsync, rename) after every change by a coalescing writer goroutine, and synchronously right after a child is spawned (so a crash cannot orphan a child). It holds, per process: id, instance, `AppConfig`, base environment, pid, process start time, command line, restarts, unstable count, created and started time, status, exit code, plus the next id.

A missing file means a fresh start. A corrupt file is renamed to `state.json.corrupt` and logged, then the daemon starts fresh.

### 4. Adoption

On start (before `--resurrect`) the daemon reads the state file. For each entry with a pid it checks that the process is alive, that its **start time** equals the recorded one (field 22 of `/proc/<pid>/stat` on Linux, `ps -o lstart=` elsewhere) and that its **command line** equals the recorded one. Only then is it adopted. A pid that exists but fails the check is a reused pid: it is never signalled, the entry is treated as "exited while the daemon was down".

An adopted process is a `child` without an `exec.Cmd`. Its exit is observed by `pidfd_open` plus `poll` on Linux (5.3+) and by polling every 500 ms (`kill(pid, 0)`, start time check, zombie check on Linux) otherwise. Its exit code is unknown (`-1`). Signals go to the process group (children are group leaders), so `stop`, `restart`, `delete`, `kill` and memory limits work as for a normal child.

An entry that was running but whose process is gone is handled by the restart policy: the restart counter is incremented and the app is started again when `autorestart` is on, otherwise it ends `errored`. Entries that were `stopped` or `errored` stay that way.

The shared listening socket (#98) lives in the children as fd 3. On Linux the daemon takes it back from an adopted child with `pidfd_getfd` and registers it again, so later restarts and reloads keep the port. Where that is not possible the app keeps running, but a restart has to bind the address again and can fail while sibling instances still hold it.

### 5. Daemon exit semantics and systemd

- `SIGTERM` and `SIGINT` mean "the daemon goes away, applications keep running": the daemon writes the state file and exits. This is what upgrades and `systemctl restart` need.
- `pm2go kill` still stops every application, removes the state file and exits.
- The generated unit uses `KillMode=process` (systemd signals only the daemon) and `Type=simple`. There is deliberately **no** `ExecStop=pm2go kill`: systemd runs `ExecStop` for `restart` as well, which would stop the applications. Run `pm2go kill` before `systemctl stop` to stop the applications too. Existing units must be regenerated with `pm2go startup`.

### Failure modes

| Case | Result |
|------|--------|
| state file missing or corrupt | fresh start, corrupt file kept as `.corrupt` |
| app died while the daemon was down | restart policy applies |
| pid reused by another program | not adopted, not signalled |
| two daemons | the existing socket check refuses the second one |
| log lines written while the daemon was down | stay in the file, are not sent to the dashboard |

## Platform notes

Linux is the production target. `/proc` start time and cmdline, `pidfd_open` and `pidfd_getfd` are in `*_linux.go` files; other platforms use `ps` and polling so the tests run on a Mac. See the PR for what was compiled but not run on Linux.

## Tests

`daemon/restart_test.go`: tail rotation and truncation, 10k lines without loss or duplicate, state file atomicity and mode, corrupt state, adoption of a hand-made orphan (same pid, counters, uptime, log lines reach the hook, stop works), delete and kill of adopted processes, an app that died while the daemon was down (restart per policy, or `errored` without autorestart), PID reuse (wrong start time, wrong command line: never adopted, never signalled), and two tests that re-execute the test binary as a real daemon: `kill -9` and restart keeps the same pids and counters, `SIGTERM` leaves the applications running, `kill` stops them and removes the state file.

## Known limits

- A child that crashes in the few milliseconds between `fork` and the first state write is orphaned and never adopted.
- Lines written while the daemon is down are in the files but not on the dashboard.
- Adopted processes have no exit code (`-1`), so `stop_exit_codes` cannot apply to the exit that ends an adopted process.
- A child that rewrites its own command line (`setproctitle`) fails the command line check and is not adopted.
- Without Linux (`pidfd_getfd`) a restart or reload of an adopted app with a shared socket must bind the address again.
