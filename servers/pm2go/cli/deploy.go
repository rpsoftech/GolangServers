package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// deployOpts are the flags of `pm2go deploy`.
type deployOpts struct {
	file     string
	dryRun   bool
	serial   bool
	failFast bool
}

// shQuote quotes s as one POSIX shell word. Single quotes switch off every
// expansion, so quotes, ';', '$()', backticks and newlines stay data.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shPath quotes a remote path, keeping a leading ~ usable ($HOME is expanded
// by the remote shell, the rest stays quoted).
func shPath(p string) string {
	switch {
	case p == "~":
		return `"$HOME"`
	case strings.HasPrefix(p, "~/"):
		return `"$HOME"` + shQuote(p[1:])
	}
	return shQuote(p)
}

const deployPrelude = `set -eu
set -o pipefail
export GIT_TERMINAL_PROMPT=0
case "$P" in /*) ;; *) P="$HOME/$P" ;; esac
export PM2GO_PATH="$P" PM2GO_SHARED="$P/shared" PM2GO_REF="$REF"
run_hook() {
  echo "--> $1"
  ( cd "$2" && bash -e -c "$3" )
}
cur_name() {
  if [ ! -L "$P/current" ]; then echo "no current release" >&2; return 1; fi
  basename "$(readlink "$P/current")"
}
switch_to() {
  link="$P/.current.$$"
  rm -f -- "$link"
  ln -s -- "$P/releases/$1" "$link"
  if ! mv -T -- "$link" "$P/current" 2>/dev/null; then mv -f -h -- "$link" "$P/current"; fi
}
`

// remoteScript builds the script sent on stdin to "bash -s". Every value is
// assigned through shQuote; nothing from the config is spliced in unquoted.
func remoteScript(cfg common.DeployConfig, cmd string, args []string) (string, error) {
	var b strings.Builder
	b.WriteString("P=" + shPath(cfg.Path) + "\n")
	b.WriteString("REF=" + shQuote(cfg.RefOrDefault()) + "\n")
	b.WriteString("REPO=" + shQuote(cfg.Repo) + "\n")
	b.WriteString("KEEP=" + strconv.Itoa(cfg.Keep()) + "\n")
	b.WriteString(deployPrelude)
	keys := make([]string, 0, len(cfg.Env))
	for k := range cfg.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString("export " + k + "=" + shQuote(cfg.Env[k]) + "\n")
	}
	hook := func(name, dir, body string) {
		if strings.TrimSpace(body) != "" {
			b.WriteString("run_hook " + name + " " + dir + " " + shQuote(body) + "\n")
		}
	}
	switch cmd {
	case "setup":
		b.WriteString(`mkdir -p -- "$P"` + "\n")
		hook("pre_setup", `"$P"`, cfg.PreSetup)
		b.WriteString(`mkdir -p -- "$P/releases" "$P/shared"` + "\n")
		b.WriteString(`if [ -d "$P/source/.git" ]; then echo "source already cloned"; else git clone --no-checkout -- "$REPO" "$P/source"; fi` + "\n")
		hook("post_setup", `"$P"`, cfg.PostSetup)
		b.WriteString(`echo "setup complete"` + "\n")
	case "update":
		b.WriteString(`[ -d "$P/source/.git" ] || { echo "not set up: run setup first" >&2; exit 1; }` + "\n")
		b.WriteString(`cd "$P/source"` + "\n")
		b.WriteString(`git fetch --prune --tags origin` + "\n")
		b.WriteString(`COMMIT=$(git rev-parse --verify "$REF^{commit}")` + "\n")
		b.WriteString(`TS=$(date -u +%Y%m%d%H%M%S)` + "\n")
		b.WriteString(`REL="$P/releases/$TS"` + "\n")
		b.WriteString(`mkdir -- "$REL"` + "\n")
		b.WriteString(`ok=0` + "\n")
		b.WriteString(`trap '[ "$ok" = 1 ] || rm -rf -- "$REL"' EXIT` + "\n")
		b.WriteString(`git archive --format=tar "$COMMIT" | tar -x -C "$REL"` + "\n")
		b.WriteString(`printf '%s\n' "$COMMIT" > "$REL/.pm2go-commit"` + "\n")
		b.WriteString(`export PM2GO_RELEASE="$REL" PM2GO_COMMIT="$COMMIT"` + "\n")
		hook("pre_deploy", `"$REL"`, cfg.PreDeploy)
		b.WriteString(`switch_to "$TS"` + "\n")
		b.WriteString(`ok=1` + "\n")
		b.WriteString(`echo "current -> $TS ($COMMIT)"` + "\n")
		hook("post_deploy", `"$P/current"`, cfg.PostDeploy)
		b.WriteString(prune)
	case "revert":
		n := 1
		if len(args) > 0 {
			v, err := strconv.Atoi(args[0])
			if err != nil || v < 1 {
				return "", fmt.Errorf("revert: n must be a positive number, got %q", args[0])
			}
			n = v
		}
		b.WriteString(`CUR=$(cur_name)` + "\n")
		b.WriteString(`TARGET=$(ls -1 "$P/releases" | sort | awk -v cur="$CUR" -v n=` + strconv.Itoa(n) + ` '{a[NR]=$0; if ($0==cur) i=NR} END { if (i-n>=1) print a[i-n] }')` + "\n")
		b.WriteString(`[ -n "$TARGET" ] || { echo "no release ` + strconv.Itoa(n) + ` before $CUR" >&2; exit 1; }` + "\n")
		b.WriteString(`switch_to "$TARGET"` + "\n")
		b.WriteString(`echo "current -> $TARGET"` + "\n")
		b.WriteString(`export PM2GO_RELEASE="$P/releases/$TARGET"` + "\n")
		hook("post_deploy", `"$P/current"`, cfg.PostDeploy)
	case "curr":
		b.WriteString(`CUR=$(cur_name)` + "\n")
		b.WriteString(`echo "$CUR $(cat "$P/releases/$CUR/.pm2go-commit" 2>/dev/null || true)"` + "\n")
	case "prev":
		b.WriteString(`CUR=$(cur_name)` + "\n")
		b.WriteString(`ls -1 "$P/releases" | sort | awk -v cur="$CUR" '{a[NR]=$0; if ($0==cur) i=NR} END { if (i>1) print a[i-1]; else exit 1 }' || { echo "no previous release" >&2; exit 1; }` + "\n")
	case "list":
		b.WriteString(`CUR=$(cur_name 2>/dev/null || true)` + "\n")
		b.WriteString(`ls -1 "$P/releases" | sort | while IFS= read -r d; do if [ "$d" = "$CUR" ]; then echo "* $d"; else echo "  $d"; fi; done` + "\n")
	case "exec":
		if len(args) == 0 || strings.TrimSpace(strings.Join(args, " ")) == "" {
			return "", errors.New(`exec needs a command: deploy <env> exec "<cmd>"`)
		}
		b.WriteString(`cd "$P/current" 2>/dev/null || cd "$P"` + "\n")
		b.WriteString("bash -c " + shQuote(strings.Join(args, " ")) + "\n")
	default:
		return "", fmt.Errorf("unknown deploy command %q (setup|update|revert|curr|prev|list|exec)", cmd)
	}
	return b.String(), nil
}

const prune = `ls -1 "$P/releases" | sort -r | awk -v k="$KEEP" 'NR>k' | while IFS= read -r d; do
  [ "$d" = "$(cur_name)" ] || rm -rf -- "$P/releases/$d"
done
`

// sshArgs is the ssh command line for one target. The script goes on stdin.
func sshArgs(cfg common.DeployConfig, target string) []string {
	args := []string{"-o", "BatchMode=yes"}
	args = append(args, cfg.SSHOptions...)
	return append(args, "--", target, "bash", "-s")
}

// lineWriter prefixes each line with "[host] " and writes it under a shared lock.
type lineWriter struct {
	mu     *sync.Mutex
	out    io.Writer
	prefix string
	buf    []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		w.emit(w.buf[:i])
		w.buf = w.buf[i+1:]
	}
	return len(p), nil
}

func (w *lineWriter) emit(line []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.out, "%s%s\n", w.prefix, strings.TrimSuffix(string(line), "\r"))
}

func (w *lineWriter) flush() {
	if len(w.buf) > 0 {
		w.emit(w.buf)
		w.buf = nil
	}
}

func cmdDeploy(args []string) error {
	var o deployOpts
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	fs.StringVar(&o.file, "f", "ecosystem.json", "ecosystem file with a deploy block")
	fs.StringVar(&o.file, "file", "ecosystem.json", "ecosystem file with a deploy block")
	fs.BoolVar(&o.dryRun, "dry-run", false, "print the steps without running them")
	fs.BoolVar(&o.serial, "serial", false, "one host at a time")
	fs.BoolVar(&o.failFast, "fail-fast", false, "stop at the first failing host")
	pos, rest, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	pos = append(pos, rest...)
	if len(pos) == 1 {
		pos = append(pos, "update") // "deploy <env>" alone means update
	}
	if len(pos) < 2 {
		return errors.New(`usage: deploy [-f file] <env> setup|update|revert [n]|curr|prev|list|exec "cmd"`)
	}
	env, sub, subArgs := pos[0], pos[1], pos[2:]
	if sub != "setup" && sub != "update" && sub != "revert" && sub != "curr" && sub != "prev" && sub != "list" && sub != "exec" {
		return fmt.Errorf("unknown deploy command %q (setup|update|revert|curr|prev|list|exec)", sub)
	}
	cfg, err := common.LoadDeploy(o.file, env)
	if err != nil {
		return err
	}
	return runDeploy(context.Background(), cfg, o, filepath.Dir(o.file), sub, subArgs, os.Stdout, os.Stderr)
}

// runDeploy validates, runs pre_deploy_local, then the script on every host.
func runDeploy(ctx context.Context, cfg common.DeployConfig, o deployOpts, dir, sub string, args []string, stdout, stderr io.Writer) error {
	if err := cfg.Validate(sub == "setup" || sub == "update"); err != nil {
		return err
	}
	script, err := remoteScript(cfg, sub, args)
	if err != nil {
		return err
	}
	if sub == "update" && strings.TrimSpace(cfg.PreDeployLocal) != "" {
		if o.dryRun {
			fmt.Fprintf(stdout, "[local] (in %s) bash -e -c %s\n", dir, shQuote(cfg.PreDeployLocal))
		} else {
			fmt.Fprintln(stdout, "[local] pre_deploy_local")
			c := exec.CommandContext(ctx, "bash", "-e", "-c", cfg.PreDeployLocal)
			c.Dir = dir
			mu := &sync.Mutex{}
			lw := &lineWriter{mu: mu, out: stdout, prefix: "[local] "}
			le := &lineWriter{mu: mu, out: stderr, prefix: "[local] "}
			c.Stdout, c.Stderr = lw, le
			err := c.Run()
			lw.flush()
			le.flush()
			if err != nil {
				return fmt.Errorf("pre_deploy_local failed: %w", err)
			}
		}
	}
	targets := cfg.Targets()
	if o.dryRun {
		for _, t := range targets {
			parts := []string{"ssh"}
			for _, a := range sshArgs(cfg, t) {
				parts = append(parts, shQuote(a))
			}
			fmt.Fprintf(stdout, "[%s] %s <<'SCRIPT'\n%s\nSCRIPT\n", t, strings.Join(parts, " "), script)
		}
		return nil
	}
	if _, err := exec.LookPath("ssh"); err != nil {
		return errors.New("ssh binary not found in PATH")
	}
	mu := &sync.Mutex{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs := make([]error, len(targets))
	run := func(i int) {
		errs[i] = runHost(ctx, cfg, targets[i], script, mu, stdout, stderr)
		if errs[i] != nil && o.failFast {
			cancel()
		}
	}
	if o.serial {
		for i := range targets {
			if ctx.Err() != nil {
				errs[i] = errors.New("skipped (fail-fast)")
				continue
			}
			run(i)
		}
	} else {
		var wg sync.WaitGroup
		for i := range targets {
			wg.Add(1)
			go func(i int) { defer wg.Done(); run(i) }(i)
		}
		wg.Wait()
	}
	var failed []string
	for i, e := range errs {
		if e != nil {
			fmt.Fprintf(stderr, "[%s] FAILED: %v\n", targets[i], e)
			failed = append(failed, targets[i])
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("deploy %s failed on %d of %d host(s): %s", sub, len(failed), len(targets), strings.Join(failed, ", "))
	}
	return nil
}

func runHost(ctx context.Context, cfg common.DeployConfig, target, script string, mu *sync.Mutex, stdout, stderr io.Writer) error {
	c := exec.CommandContext(ctx, "ssh", sshArgs(cfg, target)...)
	c.Stdin = strings.NewReader(script)
	lw := &lineWriter{mu: mu, out: stdout, prefix: "[" + target + "] "}
	le := &lineWriter{mu: mu, out: stderr, prefix: "[" + target + "] "}
	c.Stdout, c.Stderr = lw, le
	err := c.Run()
	lw.flush()
	le.flush()
	return err
}
