package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/agent"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
	"github.com/rpsoftech/golang-servers/servers/pm2go/daemon"
	"github.com/rpsoftech/golang-servers/servers/pm2go/hub"
)

const usage = `pm2go - process manager

Usage: pm2go <command> [args]

  start <script|command|ecosystem.json> [flags] [-- args...]
  stop|restart|reload|delete <id|name|namespace|all>
  scale <name> <count>    change instance count
  link --hub URL --key KEY [--name MACHINE]   connect this host to a hub
  unlink                  disconnect from the hub
  hub [--listen :8080]    run the web dashboard server
  startup [daemon|hub]    install a systemd unit (run as root)
  unstartup [daemon|hub]  remove the systemd unit
  list | ls | status      list processes
  jlist                   list processes as JSON
  describe <id|name>      show process details
  logs [name] [--lines N] [--nostream] [--out|--err]
  flush [name|all]        truncate logs
  save                    write process list to dump file
  resurrect               start processes from dump file
  ping                    check the daemon
  kill                    stop all processes and the daemon

start flags:
  --name, --cwd, --interpreter, --namespace, --env KEY=VAL (repeatable),
  --no-autorestart, --max-restarts N, --min-uptime MS, --restart-delay MS,
  --exp-backoff-restart-delay MS, --kill-timeout MS, --only a,b (ecosystem),
  --env-profile NAME (ecosystem env_<NAME>), --instances N, --watch,
  --ignore-watch a,b, --cron-restart "0 4 * * *", --max-memory-restart MB,
  --log-max-size MB (-1 off), --log-retain N, --stop-exit-codes 0,2
`

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

// parseInterspersed lets flags appear before or after positional args.
// Everything after a literal "--" is returned untouched as rest.
func parseInterspersed(fs *flag.FlagSet, args []string) (pos, rest []string, err error) {
	for len(args) > 0 {
		for i, a := range args {
			if a == "--" {
				if err := fs.Parse(args[:i]); err != nil {
					return nil, nil, err
				}
				return append(pos, fs.Args()...), args[i+1:], nil
			}
		}
		if err := fs.Parse(args); err != nil {
			return nil, nil, err
		}
		if fs.NArg() == 0 {
			break
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
	return pos, nil, nil
}

// Main runs the CLI and returns the exit code.
func Main(args []string) int {
	if len(args) == 0 {
		fmt.Print(usage)
		return 0
	}
	cmd, rest := args[0], args[1:]
	var err error
	switch cmd {
	case "daemon":
		err = cmdDaemon(rest)
	case "hub":
		err = hub.Run(rest)
	case "reload":
		err = cmdTarget("reload", rest)
	case "scale":
		err = cmdScale(rest)
	case "link":
		err = cmdLink(rest)
	case "unlink":
		err = cmdUnlink()
	case "startup":
		err = cmdStartup(rest, true)
	case "unstartup":
		err = cmdStartup(rest, false)
	case "start":
		err = cmdStart(rest)
	case "stop", "restart", "delete":
		err = cmdTarget(cmd, rest)
	case "list", "ls", "status":
		err = cmdList(false)
	case "jlist":
		err = cmdList(true)
	case "describe", "show":
		err = cmdDescribe(rest)
	case "logs":
		err = cmdLogs(rest)
	case "flush":
		err = cmdFlush(rest)
	case "save":
		var r map[string]int
		if err = call(common.MethodSave, nil, &r); err == nil {
			fmt.Printf("saved %d process(es) to %s\n", r["saved"], common.DumpFile())
		}
	case "resurrect":
		var infos []common.ProcessInfo
		if err = call(common.MethodResurrect, nil, &infos); err == nil {
			printTable(os.Stdout, infos)
		}
	case "ping":
		var r map[string]int
		if err = call(common.MethodPing, nil, &r); err == nil {
			fmt.Printf("daemon online, pid %d\n", r["pid"])
		}
	case "kill":
		err = cmdKill()
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

// startOpts holds the parsed values of the start flags.
type startOpts struct {
	name, cwd, interp, ns              string
	noAuto, watch                      bool
	maxR, minUp, delay, backoff, killT int
	only, profile                      string
	instances                          int
	ignoreWatch, cronR                 string
	maxMem, logMax, logRetain          int
	stopCodes                          string
	envs                               multi
}

// newStartFlagSet declares the start flags and returns the values they fill.
func newStartFlagSet() (*flag.FlagSet, *startOpts) {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	o := &startOpts{}
	fs.StringVar(&o.name, "name", "", "process name")
	fs.StringVar(&o.cwd, "cwd", "", "working directory")
	fs.StringVar(&o.interp, "interpreter", "", "interpreter binary")
	fs.StringVar(&o.ns, "namespace", "", "namespace")
	fs.BoolVar(&o.noAuto, "no-autorestart", false, "disable auto restart")
	fs.IntVar(&o.maxR, "max-restarts", 0, "max consecutive unstable restarts")
	fs.IntVar(&o.minUp, "min-uptime", 0, "ms a run must last to count as stable")
	fs.IntVar(&o.delay, "restart-delay", 0, "ms between restarts")
	fs.IntVar(&o.backoff, "exp-backoff-restart-delay", 0, "base ms for exponential backoff")
	fs.IntVar(&o.killT, "kill-timeout", 0, "ms before SIGKILL")
	fs.StringVar(&o.only, "only", "", "comma separated app names (ecosystem)")
	fs.StringVar(&o.profile, "env-profile", "", "ecosystem env_<name> block")
	fs.IntVar(&o.instances, "instances", 0, "number of instances")
	fs.BoolVar(&o.watch, "watch", false, "restart on file change")
	fs.StringVar(&o.ignoreWatch, "ignore-watch", "", "comma separated ignore patterns")
	fs.StringVar(&o.cronR, "cron-restart", "", "cron spec for periodic restart")
	fs.IntVar(&o.maxMem, "max-memory-restart", 0, "restart above this many MB")
	fs.IntVar(&o.logMax, "log-max-size", 0, "rotate logs at this many MB (-1 off)")
	fs.IntVar(&o.logRetain, "log-retain", 0, "rotated log files to keep")
	fs.StringVar(&o.stopCodes, "stop-exit-codes", "", "exit codes that do not restart")
	fs.Var(&o.envs, "env", "KEY=VAL (repeatable)")
	return fs, o
}

// resolveStartTargets turns the start target into the app configs to start:
// every app of an ecosystem file (filtered by --only) or one command.
func resolveStartTargets(o *startOpts, target string, scriptArgs []string) ([]common.AppConfig, error) {
	if !strings.HasSuffix(strings.ToLower(target), ".json") {
		c, err := buildStartConfig(o, target, scriptArgs)
		if err != nil {
			return nil, err
		}
		return []common.AppConfig{c}, nil
	}
	all, err := common.LoadEcosystem(target, o.profile)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, n := range strings.Split(o.only, ",") {
		if n = strings.TrimSpace(n); n != "" {
			want[n] = true
		}
	}
	var cfgs []common.AppConfig
	for _, c := range all {
		if len(want) == 0 || want[c.Name] {
			cfgs = append(cfgs, c)
		}
	}
	if len(cfgs) == 0 {
		return nil, errors.New("no app matches --only")
	}
	return cfgs, nil
}

// buildStartConfig builds the config of one command from the start flags.
func buildStartConfig(o *startOpts, target string, scriptArgs []string) (common.AppConfig, error) {
	var err error
	c := common.AppConfig{
		Name: o.name, Script: target, Args: scriptArgs, Cwd: o.cwd, Interpreter: o.interp,
		Namespace: o.ns, MaxRestarts: o.maxR, MinUptimeMs: o.minUp, RestartDelayMs: o.delay,
		ExpBackoffRestartDelayMs: o.backoff, KillTimeoutMs: o.killT,
	}
	if c.Name == "" {
		c.Name = common.DefaultName(target)
	}
	if c.Cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return c, err
		}
		c.Cwd = wd
	} else if c.Cwd, err = filepath.Abs(c.Cwd); err != nil {
		return c, err
	}
	c.Instances, c.Watch, c.CronRestart = o.instances, o.watch, o.cronR
	c.MaxMemoryRestartMB, c.LogMaxSizeMB, c.LogRetain = o.maxMem, o.logMax, o.logRetain
	for _, x := range strings.Split(o.ignoreWatch, ",") {
		if x = strings.TrimSpace(x); x != "" {
			c.IgnoreWatch = append(c.IgnoreWatch, x)
		}
	}
	for _, x := range strings.Split(o.stopCodes, ",") {
		if x = strings.TrimSpace(x); x != "" {
			n, err := strconv.Atoi(x)
			if err != nil {
				return c, fmt.Errorf("bad --stop-exit-codes value %q", x)
			}
			c.StopExitCodes = append(c.StopExitCodes, n)
		}
	}
	if o.noAuto {
		f := false
		c.Autorestart = &f
	}
	if len(o.envs) > 0 {
		c.Env = map[string]string{}
		for _, kv := range o.envs {
			k, v, ok := strings.Cut(kv, "=")
			if !ok || k == "" {
				return c, fmt.Errorf("bad --env %q, want KEY=VAL", kv)
			}
			c.Env[k] = v
		}
	}
	return c, nil
}

func cmdStart(args []string) error {
	fs, o := newStartFlagSet()
	pos, scriptArgs, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("start needs exactly one script, command or ecosystem file")
	}
	cfgs, err := resolveStartTargets(o, pos[0], scriptArgs)
	if err != nil {
		return err
	}

	var all []common.ProcessInfo
	for _, c := range cfgs {
		var infos []common.ProcessInfo
		if err := call(common.MethodStart, common.StartParams{Config: c, CLIEnv: os.Environ()}, &infos); err != nil {
			return fmt.Errorf("%s: %w", c.Name, err)
		}
		all = append(all, infos...)
	}
	return listAfter(all)
}

// listAfter prints the current table so users see state after a change.
func listAfter(_ []common.ProcessInfo) error { return cmdList(false) }

func cmdTarget(method string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%s needs one target: id, name, namespace or all", method)
	}
	var infos []common.ProcessInfo
	if err := call(method, common.TargetParams{Target: args[0]}, &infos); err != nil {
		return err
	}
	return cmdList(false)
}

func cmdList(asJSON bool) error {
	var infos []common.ProcessInfo
	if err := call(common.MethodList, nil, &infos); err != nil {
		return err
	}
	if asJSON {
		b, _ := json.MarshalIndent(infos, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	printTable(os.Stdout, infos)
	return nil
}

func cmdDescribe(args []string) error {
	if len(args) != 1 {
		return errors.New("describe needs one target")
	}
	var infos []common.ProcessInfo
	if err := call(common.MethodDescribe, common.TargetParams{Target: args[0]}, &infos); err != nil {
		return err
	}
	for n, i := range infos {
		if n > 0 {
			fmt.Println()
		}
		printDescribe(os.Stdout, i)
	}
	return nil
}

func cmdLogs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	lines := fs.Int("lines", 15, "lines to show per file")
	nostream := fs.Bool("nostream", false, "print and exit")
	outOnly := fs.Bool("out", false, "stdout only")
	errOnly := fs.Bool("err", false, "stderr only")
	pos, _, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	target := "all"
	if len(pos) > 0 {
		target = pos[0]
	}
	var infos []common.ProcessInfo
	if err := call(common.MethodDescribe, common.TargetParams{Target: target}, &infos); err != nil {
		return err
	}
	showLogs(infos, *lines, !*nostream, *outOnly, *errOnly)
	return nil
}

func cmdFlush(args []string) error {
	target := "all"
	if len(args) > 0 {
		target = args[0]
	}
	return call(common.MethodFlush, common.TargetParams{Target: target}, nil)
}

func cmdKill() error {
	conn, err := connect(false)
	if err != nil {
		fmt.Println("daemon is not running")
		return nil
	}
	defer conn.Close()
	if err := roundTrip(conn, common.MethodKill, nil, nil); err != nil {
		return err
	}
	for i := 0; i < 200; i++ { // wait for children to be stopped
		c, err := dial()
		if err != nil {
			break
		}
		c.Close()
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println("daemon stopped")
	return nil
}

func cmdScale(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: scale <name> <count>")
	}
	n, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("bad count %q", args[1])
	}
	if err := call(common.MethodScale, common.ScaleParams{Name: args[0], Count: n}, nil); err != nil {
		return err
	}
	return cmdList(false)
}

func cmdDaemon(args []string) error {
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	res := fs.Bool("resurrect", false, "start saved processes on boot")
	if _, _, err := parseInterspersed(fs, args); err != nil {
		return err
	}
	return daemon.Run(daemon.Options{Resurrect: *res, OnReady: func(m *daemon.Manager) (func(), func()) {
		a := agent.New(m, Version)
		a.Reload()
		return a.Reload, a.Stop
	}})
}

func cmdLink(args []string) error {
	fs := flag.NewFlagSet("link", flag.ContinueOnError)
	hub := fs.String("hub", "", "hub URL, e.g. http://10.0.0.5:8080")
	key := fs.String("key", "", "agent key printed by the hub")
	name := fs.String("name", "", "machine name (default: hostname)")
	if _, _, err := parseInterspersed(fs, args); err != nil {
		return err
	}
	if *hub == "" || *key == "" {
		return errors.New("link needs --hub and --key")
	}
	if err := common.SaveLink(common.LinkConfig{HubURL: *hub, Key: *key, Machine: *name}); err != nil {
		return err
	}
	if err := call(common.MethodLink, nil, nil); err != nil {
		return err
	}
	fmt.Println("linked to", *hub)
	return nil
}

func cmdUnlink() error {
	if err := os.Remove(common.LinkFile()); err != nil && !os.IsNotExist(err) {
		return err
	}
	if conn, err := connect(false); err == nil {
		defer conn.Close()
		_ = roundTrip(conn, common.MethodLink, nil, nil)
	}
	fmt.Println("unlinked")
	return nil
}
