# pm2go monit

`pm2go monit` is a full-screen terminal dashboard (issue #94). It uses only the standard library plus `golang.org/x/term` (raw mode and size). Code: `servers/pm2go/cli/monit.go` (logic, render, loop), `monit_unix.go` / `monit_windows.go` (resize signal).

## What it shows

- Row 1: host summary (online count, cpu, memory, load, uptime) from the `system` RPC.
- Process list: id, name, status (coloured), cpu %, cpu bar, memory. Bars are dropped below 52 columns.
- Log pane: last 200 lines of the selected process (stdout, or stderr with `e`), re-read every second with `lastLines` from `cli/logs.go`.
- Last row: key help, a filter or delete prompt, or the last action result / error.
- Width >= 100: list left (45 %, max 70 columns), logs right. Narrower: list on top (up to half the body), logs below. Smaller than 30x6: a "terminal too small" line.

## Keys

`up`/`down` (or `k`/`j`) select, `r` restart, `s` stop, `d` delete (asks `y/n`), `f` flush logs, `e` stdout/stderr, `/` filter by name (`enter` keeps it, `esc` clears), `q` or `ctrl-c` quit. Actions target the numeric process id, so a name shared by several instances acts on the selected instance only.

## Design decisions

- `render(state, w, h)` and `monitState.key(k)` are pure; `runMonit` takes its input, output, size, resize, quit and tick as channels / funcs, so tests run the loop without a TTY.
- One frame = one `Write`: cursor home, every row followed by `ESC[K`, then `ESC[J`. Nothing is cleared first, so no flicker. Raw mode turns off output post-processing, hence `\r\n`.
- The terminal is restored (leave alternate screen, show cursor, reset colours, cooked mode) by a `sync.Once` called from a `defer`, from a `recover` that re-panics, and on SIGINT / SIGTERM / SIGHUP. `ctrl-c` also arrives as byte 3 because raw mode disables ISIG.
- No TTY on stdin or stdout: error `monit needs an interactive terminal ...`, exit 1. Daemon not running: error, exit 1, the daemon is never auto-spawned. If the daemon dies while running, the last data stays and the status line shows the error until it returns.
- Every RPC uses a fresh connection with a 5 s deadline. Actions run in a goroutine so a slow stop does not freeze the screen.
- Log lines are stripped of escape sequences, tabs and control characters before drawing, so a noisy app cannot corrupt the screen. Wide (CJK) characters are counted as one cell.
- New RPC `system` (`common.MethodSystem`) returns `Manager.System()` (`SystemSample`). It is additive; an older daemon answers `unknown method` and monit then hides the host figures.
- `NO_COLOR` set: no colours, selection shown with `>`.
- Windows has no SIGWINCH: the size is re-read on the one-second refresh.
- `golang.org/x/term` was an indirect dependency; the only `go.mod` change is dropping its `// indirect` comment.

## Tests

`cli/monit_test.go` (render at several sizes, content, key handling, key parsing, loop with a fake daemon, no TTY, daemon down) and `cli/monit_system_test.go` (real in-process daemon: `list` + `system`).
