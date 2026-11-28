package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// fakeSSH installs a fake ssh first in PATH. It records its arguments and
// stdin per call in <log>/N.args and N.stdin. With FAKE_SSH_EXEC=1 it runs the
// stdin script locally with bash (the host is ignored), so the real script,
// real git and a real filesystem are exercised. FAKE_SSH_FAIL=<target> makes
// that target exit 1 without running anything.
func fakeSSH(t *testing.T, execLocal bool) (logDir string) {
	t.Helper()
	bin, logDir := t.TempDir(), t.TempDir()
	script := `#!/bin/bash
n=$(ls "$FAKE_LOG" | grep -c args || true)
n=$((n+1))
mkdir -p "$FAKE_LOG"
printf '%s\n' "$@" > "$FAKE_LOG/$$.args"
cat > "$FAKE_LOG/$$.stdin"
for a in "$@"; do
  if [ "$a" = "$FAKE_SSH_FAIL" ]; then echo "fake ssh: connection refused" >&2; exit 255; fi
done
if [ "$FAKE_SSH_EXEC" = 1 ]; then bash -s < "$FAKE_LOG/$$.stdin"; fi
`
	if err := os.WriteFile(filepath.Join(bin, "ssh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_LOG", logDir)
	t.Setenv("FAKE_SSH_FAIL", "")
	if execLocal {
		t.Setenv("FAKE_SSH_EXEC", "1")
	} else {
		t.Setenv("FAKE_SSH_EXEC", "")
	}
	return logDir
}

type sshCall struct {
	args  []string
	stdin string
}

func readCalls(t *testing.T, dir string) []sshCall {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(dir, "*.args"))
	var out []sshCall
	for _, f := range files {
		a, _ := os.ReadFile(f)
		s, _ := os.ReadFile(strings.TrimSuffix(f, ".args") + ".stdin")
		out = append(out, sshCall{strings.Split(strings.TrimSuffix(string(a), "\n"), "\n"), string(s)})
	}
	return out
}

func sh(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command(args[0], args[1:]...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func deploy(t *testing.T, cfg common.DeployConfig, o deployOpts, sub string, args ...string) (string, string, error) {
	t.Helper()
	var so, se bytes.Buffer
	err := runDeploy(context.Background(), cfg, o, t.TempDir(), sub, args, &so, &se)
	return so.String(), se.String(), err
}

func baseCfg(path string) common.DeployConfig {
	return common.DeployConfig{User: "app", Host: common.StringList{"h1.example.com"}, Repo: "git@example.com:o/r.git", Path: path}
}

func TestShQuote(t *testing.T) {
	cases := map[string]string{
		"plain":      "'plain'",
		"it's":       `'it'\''s'`,
		"a;b":        "'a;b'",
		"$(id)":      "'$(id)'",
		"`id`":       "'`id`'",
		"a\nb":       "'a\nb'",
		`"q"`:        `'"q"'`,
		"":           "''",
		"x'; rm -rf": `'x'\''; rm -rf'`,
	}
	for in, want := range cases {
		if got := shQuote(in); got != want {
			t.Errorf("shQuote(%q) = %s, want %s", in, got, want)
		}
	}
	if got := shPath("~/apps/x y"); got != `"$HOME"'/apps/x y'` {
		t.Errorf("shPath tilde: %s", got)
	}
}

func TestDeployRecordsSSHArgsAndStdin(t *testing.T) {
	log := fakeSSH(t, false)
	cfg := baseCfg("/srv/app")
	cfg.Host = common.StringList{"h1", "h2"}
	cfg.SSHOptions = []string{"-p", "2222"}
	cfg.PostDeploy = "pm2go reload ecosystem.json"
	if _, _, err := deploy(t, cfg, deployOpts{serial: true}, "update"); err != nil {
		t.Fatal(err)
	}
	calls := readCalls(t, log)
	if len(calls) != 2 {
		t.Fatalf("want 2 ssh calls, got %d", len(calls))
	}
	for _, c := range calls {
		want := []string{"-o", "BatchMode=yes", "-p", "2222", "--"}
		for i, w := range want {
			if c.args[i] != w {
				t.Fatalf("arg %d = %q want %q (%v)", i, c.args[i], w, c.args)
			}
		}
		tail := c.args[len(c.args)-3:]
		if tail[1] != "bash" || tail[2] != "-s" || !strings.HasPrefix(tail[0], "app@h") {
			t.Fatalf("bad tail %v", tail)
		}
		for _, a := range c.args {
			if strings.Contains(a, "reload") || strings.Contains(a, "/srv/app") {
				t.Fatalf("config value leaked into argv: %v", c.args)
			}
		}
		if !strings.Contains(c.stdin, "pm2go reload ecosystem.json") || !strings.Contains(c.stdin, "P='/srv/app'") {
			t.Fatalf("script missing values:\n%s", c.stdin)
		}
	}
}

func TestDefaultsKeepHostKeyChecks(t *testing.T) {
	log := fakeSSH(t, false)
	if _, _, err := deploy(t, baseCfg("/srv/app"), deployOpts{}, "curr"); err != nil {
		t.Fatal(err)
	}
	for _, c := range readCalls(t, log) {
		if strings.Contains(strings.Join(c.args, " "), "StrictHostKeyChecking") {
			t.Fatalf("host key checking must not be touched by default: %v", c.args)
		}
	}
}

func TestDryRunDoesNotRunSSH(t *testing.T) {
	log := fakeSSH(t, false)
	cfg := baseCfg("/srv/app")
	cfg.PreDeployLocal = "touch " + filepath.Join(t.TempDir(), "never")
	out, _, err := deploy(t, cfg, deployOpts{dryRun: true}, "update")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(readCalls(t, log)); n != 0 {
		t.Fatalf("dry run called ssh %d times", n)
	}
	for _, w := range []string{"[app@h1.example.com]", "'BatchMode=yes'", "git archive", "[local]"} {
		if !strings.Contains(out, w) {
			t.Errorf("dry-run output missing %q:\n%s", w, out)
		}
	}
}

func TestInjectionPayloadsStayQuoted(t *testing.T) {
	log := fakeSSH(t, false)
	marker := filepath.Join(t.TempDir(), "pwned")
	payloads := []string{
		"x'; touch " + marker + "; echo '",
		"$(touch " + marker + ")",
		"`touch " + marker + "`",
		"a;touch " + marker,
		"a\ntouch " + marker + "\n",
		`"; touch ` + marker + `; "`,
	}
	for _, p := range payloads {
		cfg := baseCfg("/srv/" + p)
		cfg.Ref = "origin/" + p
		cfg.Repo = "git@h:" + p
		cfg.PreDeploy = p
		cfg.Env = map[string]string{"K": p}
		for _, sub := range []string{"setup", "update"} {
			if _, _, err := deploy(t, cfg, deployOpts{}, sub); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, c := range readCalls(t, log) {
		for _, a := range c.args {
			if strings.Contains(a, "touch") {
				t.Fatalf("payload reached argv: %v", c.args)
			}
		}
		if !strings.Contains(c.stdin, "REF="+shQuote(refOf(c.stdin))) {
			t.Fatalf("ref not quoted:\n%s", c.stdin)
		}
		if _, err := os.Stat(marker); err == nil {
			t.Fatal("payload executed")
		}
	}
}

// With the exec fake, run the generated scripts for real and check that
// payloads in path, ref, repo and hooks never run as part of the script.
func TestInjectionNotExecutedForReal(t *testing.T) {
	fakeSSH(t, true)
	root := t.TempDir()
	marker := filepath.Join(root, "pwned")
	payloads := []string{
		"x'; touch " + marker + "; echo '",
		"$(touch " + marker + ")",
		"`touch " + marker + "`",
		"a;touch " + marker,
		"a\ntouch " + marker + "\nb",
	}
	for i, p := range payloads {
		cfg := baseCfg(filepath.Join(root, fmt.Sprintf("p%d-", i)+p))
		cfg.Repo = "/nonexistent/" + p
		cfg.Ref = "origin/" + p
		cfg.Env = map[string]string{"K": p}
		// setup fails on the clone (bad repo); update fails (not set up).
		deploy(t, cfg, deployOpts{}, "setup")
		deploy(t, cfg, deployOpts{}, "update")
		deploy(t, cfg, deployOpts{}, "exec", "echo hi")
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("payload %q executed", p)
		}
	}
}

func TestHookValueIsOneArgument(t *testing.T) {
	fakeSSH(t, true)
	root := t.TempDir()
	marker := filepath.Join(root, "hookran")
	cfg := baseCfg(filepath.Join(root, "app"))
	// A hook runs, but a quote inside it cannot break out of the script: the
	// rest of the script still runs after it.
	cfg.PostSetup = `echo "it's"; touch '` + marker + `'`
	cfg.Repo = mkRepo(t, "v1")
	if _, _, err := deploy(t, cfg, deployOpts{}, "setup"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("hook did not run")
	}
}

func TestValidate(t *testing.T) {
	bad := []func(*common.DeployConfig){
		func(c *common.DeployConfig) { c.Host = common.StringList{"-oProxyCommand=x"} },
		func(c *common.DeployConfig) { c.Host = common.StringList{"a b"} },
		func(c *common.DeployConfig) { c.Host = common.StringList{"h;id"} },
		func(c *common.DeployConfig) { c.User = "-x" },
		func(c *common.DeployConfig) { c.Ref = "--upload-pack=x" },
		func(c *common.DeployConfig) { c.Repo = "--upload-pack=x" },
		func(c *common.DeployConfig) { c.Path = "/" },
		func(c *common.DeployConfig) { c.Env = map[string]string{"A;B": "1"} },
		func(c *common.DeployConfig) { c.Path = "/a\x00b" },
	}
	for i, f := range bad {
		c := baseCfg("/srv/app")
		f(&c)
		if err := c.Validate(true); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}

func TestFailureReportsHostAndOthersContinue(t *testing.T) {
	log := fakeSSH(t, false)
	t.Setenv("FAKE_SSH_FAIL", "bad")
	cfg := baseCfg("/srv/app")
	cfg.User = ""
	cfg.Host = common.StringList{"bad", "good1", "good2"}
	_, se, err := deploy(t, cfg, deployOpts{serial: true}, "curr")
	if err == nil || !strings.Contains(err.Error(), "bad") || !strings.Contains(err.Error(), "1 of 3") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(se, "[bad] FAILED") {
		t.Errorf("stderr: %s", se)
	}
	if n := len(readCalls(t, log)); n != 3 {
		t.Fatalf("others must continue: %d ssh calls", n)
	}
}

func TestFailFastSerialSkipsRest(t *testing.T) {
	log := fakeSSH(t, false)
	t.Setenv("FAKE_SSH_FAIL", "bad")
	cfg := baseCfg("/srv/app")
	cfg.User = ""
	cfg.Host = common.StringList{"bad", "good1", "good2"}
	_, _, err := deploy(t, cfg, deployOpts{serial: true, failFast: true}, "curr")
	if err == nil {
		t.Fatal("want error")
	}
	if n := len(readCalls(t, log)); n != 1 {
		t.Fatalf("fail-fast must stop: %d ssh calls", n)
	}
}

func TestOutputPrefixedPerHost(t *testing.T) {
	fakeSSH(t, true)
	root := t.TempDir()
	cfg := baseCfg(filepath.Join(root, "app"))
	cfg.Repo = mkRepo(t, "v1")
	cfg.Host = common.StringList{"h1", "h2"}
	cfg.User = ""
	cfg.PostSetup = "echo hello"
	out, _, err := deploy(t, cfg, deployOpts{serial: true}, "setup")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, "[h1] setup complete") != 1 || strings.Count(out, "[h2] setup complete") != 1 {
		t.Fatalf("output not prefixed per host:\n%s", out)
	}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.HasPrefix(l, "[h1] ") && !strings.HasPrefix(l, "[h2] ") {
			t.Fatalf("unprefixed line %q", l)
		}
	}
}

// mkRepo creates a git repo with one commit holding VERSION=<v>.
func mkRepo(t *testing.T, v string) string {
	t.Helper()
	dir := t.TempDir()
	sh(t, dir, "git", "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte(v+"\n"), 0o644)
	sh(t, dir, "git", "add", ".")
	sh(t, dir, "git", "commit", "-q", "-m", v)
	return dir
}

func commitVersion(t *testing.T, repo, v string) {
	t.Helper()
	os.WriteFile(filepath.Join(repo, "VERSION"), []byte(v+"\n"), 0o644)
	sh(t, repo, "git", "commit", "-q", "-am", v)
}

func TestFullWorkflowAgainstLocalShell(t *testing.T) {
	fakeSSH(t, true)
	root := t.TempDir()
	repo := mkRepo(t, "v1")
	app := filepath.Join(root, "app")
	cfg := baseCfg(app)
	cfg.Repo = repo
	cfg.KeepReleases = 2
	cfg.PreSetup = "echo pre-setup"
	cfg.PostSetup = "touch \"$PM2GO_SHARED/setup-done\""
	cfg.PostDeploy = "cp VERSION \"$PM2GO_SHARED/deployed\""
	cfg.Env = map[string]string{"GREETING": "it's $fine"}
	cfg.PreDeploy = `test "$GREETING" = "it's \$fine" && test -f VERSION`

	out, _, err := deploy(t, cfg, deployOpts{}, "setup")
	if err != nil {
		t.Fatalf("setup: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(app, "shared", "setup-done")); err != nil {
		t.Fatal("post_setup did not run")
	}
	readLink := func() string {
		l, err := os.Readlink(filepath.Join(app, "current"))
		if err != nil {
			t.Fatal(err)
		}
		return filepath.Base(l)
	}
	var rels []string
	for _, v := range []string{"v1", "v2", "v3"} {
		if v != "v1" {
			commitVersion(t, repo, v)
		}
		// Release names are second-resolution timestamps: wait for a new one.
		if len(rels) > 0 {
			waitNewSecond(t)
		}
		if out, _, err := deploy(t, cfg, deployOpts{}, "update"); err != nil {
			t.Fatalf("update %s: %v\n%s", v, err, out)
		}
		rels = append(rels, readLink())
		b, _ := os.ReadFile(filepath.Join(app, "shared", "deployed"))
		if strings.TrimSpace(string(b)) != v {
			t.Fatalf("post_deploy ran in wrong dir: got %q want %s", b, v)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(app, "releases"))
	if len(entries) != 2 {
		t.Fatalf("keep_releases=2 but %d releases remain", len(entries))
	}
	if _, err := os.Stat(filepath.Join(app, "releases", rels[0])); err == nil {
		t.Fatal("oldest release not pruned")
	}

	out, _, _ = deploy(t, cfg, deployOpts{}, "curr")
	if !strings.Contains(out, rels[2]) {
		t.Errorf("curr: %s", out)
	}
	out, _, _ = deploy(t, cfg, deployOpts{}, "prev")
	if !strings.Contains(out, rels[1]) {
		t.Errorf("prev: %s", out)
	}
	out, _, _ = deploy(t, cfg, deployOpts{}, "list")
	if !strings.Contains(out, "* "+rels[2]) || !strings.Contains(out, "  "+rels[1]) {
		t.Errorf("list: %s", out)
	}
	if out, _, err := deploy(t, cfg, deployOpts{}, "revert"); err != nil {
		t.Fatalf("revert: %v\n%s", err, out)
	}
	if readLink() != rels[1] {
		t.Fatalf("revert did not switch: %s", readLink())
	}
	b, _ := os.ReadFile(filepath.Join(app, "shared", "deployed"))
	if strings.TrimSpace(string(b)) != "v2" {
		t.Fatalf("post_deploy not re-run on revert: %q", b)
	}
	if _, _, err := deploy(t, cfg, deployOpts{}, "revert"); err == nil {
		t.Fatal("revert past the oldest release must fail")
	}
	out, _, err = deploy(t, cfg, deployOpts{}, "exec", "cat VERSION")
	if err != nil || !strings.Contains(out, "v2") {
		t.Fatalf("exec: %v %s", err, out)
	}
}

func TestFailedPreDeployKeepsCurrent(t *testing.T) {
	fakeSSH(t, true)
	root := t.TempDir()
	repo := mkRepo(t, "v1")
	app := filepath.Join(root, "app")
	cfg := baseCfg(app)
	cfg.Repo = repo
	if _, _, err := deploy(t, cfg, deployOpts{}, "setup"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := deploy(t, cfg, deployOpts{}, "update"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Readlink(filepath.Join(app, "current"))
	waitNewSecond(t)
	commitVersion(t, repo, "v2")
	cfg.PreDeploy = "exit 3"
	if _, _, err := deploy(t, cfg, deployOpts{}, "update"); err == nil {
		t.Fatal("want failure")
	}
	after, _ := os.Readlink(filepath.Join(app, "current"))
	if before != after {
		t.Fatalf("current moved despite failed pre_deploy: %s -> %s", before, after)
	}
	entries, _ := os.ReadDir(filepath.Join(app, "releases"))
	if len(entries) != 1 {
		t.Fatalf("half built release left behind: %d", len(entries))
	}
}

func TestPreDeployLocalRunsAndAbortsOnFailure(t *testing.T) {
	log := fakeSSH(t, false)
	dir := t.TempDir()
	marker := filepath.Join(dir, "local")
	cfg := baseCfg("/srv/app")
	cfg.PreDeployLocal = "touch " + marker
	var so, se bytes.Buffer
	if err := runDeploy(context.Background(), cfg, deployOpts{}, dir, "update", nil, &so, &se); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("pre_deploy_local did not run")
	}
	cfg.PreDeployLocal = "exit 1"
	before := len(readCalls(t, log))
	if err := runDeploy(context.Background(), cfg, deployOpts{}, dir, "update", nil, &so, &se); err == nil {
		t.Fatal("want error")
	}
	if len(readCalls(t, log)) != before {
		t.Fatal("ssh ran after failed pre_deploy_local")
	}
}

func TestLoadDeployJSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "ecosystem.json")
	doc := map[string]any{
		"apps": []map[string]any{{"name": "a", "script": "./a"}},
		"deploy": map[string]any{"prod": map[string]any{
			"user": "app", "host": []string{"h1", "h2"}, "path": "/srv/a", "repo": "r", "keep_releases": 3,
		}, "stage": map[string]any{"host": "one", "path": "/s"}},
	}
	b, _ := json.Marshal(doc)
	os.WriteFile(f, b, 0o644)
	cfg, err := common.LoadDeploy(f, "prod")
	if err != nil || len(cfg.Host) != 2 || cfg.Keep() != 3 || cfg.RefOrDefault() != "origin/main" {
		t.Fatalf("%+v %v", cfg, err)
	}
	st, err := common.LoadDeploy(f, "stage")
	if err != nil || len(st.Host) != 1 || st.Keep() != 5 {
		t.Fatalf("%+v %v", st, err)
	}
	if _, err := common.LoadDeploy(f, "nope"); err == nil {
		t.Fatal("want error for unknown env")
	}
}

func TestCmdDeployUsageErrors(t *testing.T) {
	if err := cmdDeploy(nil); err == nil {
		t.Fatal("want usage error")
	}
	f := filepath.Join(t.TempDir(), "e.json")
	os.WriteFile(f, []byte(`{"deploy":{"p":{"host":"h","path":"/x","repo":"r"}}}`), 0o644)
	if err := cmdDeploy([]string{"-f", f, "p", "bogus"}); err == nil || !strings.Contains(err.Error(), "unknown deploy command") {
		t.Fatalf("err = %v", err)
	}
	fakeSSH(t, false)
	out, _ := capture(t, func() {
		if err := cmdDeploy([]string{"p", "--dry-run", "-f", f}); err != nil {
			t.Error(err)
		}
	})
	if !strings.Contains(out, "git archive") {
		t.Errorf("bare <env> should run update: %s", out)
	}
}

func waitNewSecond(t *testing.T) {
	t.Helper()
	time.Sleep(1100 * time.Millisecond)
}

// refOf returns the ref of a recorded script: the payload sits between
// "REF='" and the line that starts "REPO=".
func refOf(script string) string {
	i := strings.Index(script, "REF=")
	j := strings.Index(script, "\nREPO=")
	q := script[i+4 : j]
	return strings.ReplaceAll(q[1:len(q)-1], `'\''`, "'")
}
