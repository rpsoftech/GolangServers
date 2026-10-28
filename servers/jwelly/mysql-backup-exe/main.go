package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/robfig/cron/v3"
	"github.com/rpsoftech/golang-servers/utility/updater"
)

var version = "1" // Injected via -ldflags during build

func main() {
	var (
		daemon     = flag.Bool("daemon", false, "run in the background and back up on each database's cron schedule")
		configPath = flag.String("config", "", "path to the config file (default: config.json next to the executable)")
		noPause    = flag.Bool("no-pause", false, "do not wait for Enter before closing (quick run)")
		_          = flag.Bool("dev", false, "accepted for compatibility with other tools; no effect")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags)
	notifier := newNotifier()

	cfgFile := resolveConfigPath(*configPath)
	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		if *daemon {
			notifier.Notify("Backup config error", firstLine(err.Error()))
		}
		finish(1, *daemon || *noPause)
		return
	}

	if *daemon {
		runDaemon(cfg, notifier)
		return
	}
	finish(runQuick(cfg, notifier), *noPause)
}

// resolveConfigPath uses --config if given, otherwise config.json next to the
// executable, falling back to the working directory.
func resolveConfigPath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "config.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "config.json"
}

// finish exits, first waiting for Enter when the console window would
// otherwise close before the user can read the result.
func finish(code int, skipPause bool) {
	if !skipPause && ownsConsole() {
		fmt.Print("\nPress Enter to close...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
	os.Exit(code)
}

// --- quick run ---

// runQuick backs up every configured database once and prints a summary. It
// returns the process exit code.
func runQuick(cfg *Config, notifier Notifier) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var results []*Result
	for _, db := range cfg.Databases {
		results = append(results, RunBackup(ctx, db, notifier))
		if ctx.Err() != nil {
			break
		}
	}

	failed := 0
	fmt.Println("\n=== Backup summary ===")
	for _, r := range results {
		if r.OK() {
			fmt.Printf("OK     %-20s %s via %s in %s\n", r.Name, humanSize(r.Size), r.Engine, r.Duration.Round(100*1e6))
		} else {
			failed++
			fmt.Printf("FAILED %s\n", r.Name)
		}
		if r.Err != nil {
			fmt.Printf("         dump: %s\n", indent(r.Err.Error()))
		}
		for _, d := range r.Dests {
			if d.Err != nil {
				fmt.Printf("         %s: %s\n", d.Label, indent(d.Err.Error()))
			} else if r.Err == nil {
				fmt.Printf("         -> %s\n", d.Label)
			}
		}
	}
	if skipped := len(cfg.Databases) - len(results); skipped > 0 {
		fmt.Printf("INTERRUPTED: %d database(s) not backed up\n", skipped)
		failed += skipped
	}
	if failed > 0 {
		fmt.Printf("\n%d of %d backup(s) failed.\n", failed, len(cfg.Databases))
		return 1
	}
	fmt.Printf("\nAll %d backup(s) completed.\n", len(results))
	return 0
}

func indent(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", "\n           ")
}

// --- daemon ---

func runDaemon(cfg *Config, notifier Notifier) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// OTA updates: only published builds update, and only from their own
	// release channel. A successful update cancels ctx to restart cleanly.
	go updater.RunDaemon(ctx, updater.Config{
		Component:      updater.MysqlBackupExeProjectName,
		CurrentVersion: updater.ParseVersion(version),
	}, cancel)

	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DiscardLogger)))
	scheduled := 0
	for _, db := range cfg.Databases {
		if db.Cron == "" {
			log.Printf("[%s] no cron set, skipped in daemon mode", db.Name)
			continue
		}
		if _, err := c.AddFunc(db.Cron, func() {
			res := RunBackup(ctx, db, notifier)
			reportDaemonResult(res, notifier)
		}); err != nil {
			log.Printf("[%s] cannot schedule %q: %v", db.Name, db.Cron, err)
			continue
		}
		scheduled++
		log.Printf("[%s] scheduled: %s", db.Name, db.Cron)
	}
	if scheduled == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: no database has a valid cron schedule; nothing to run in daemon mode")
		notifier.Notify("Backup daemon not started", "No database has a cron schedule.")
		os.Exit(1)
	}

	c.Start()
	log.Println("backup daemon started")
	<-ctx.Done()
	log.Println("shutting down, waiting for running backups to finish...")
	<-c.Stop().Done()
	log.Println("mysql-backup-exe stopped")
}

func reportDaemonResult(r *Result, notifier Notifier) {
	if r.OK() {
		log.Printf("[%s] backup completed (%s via %s)", r.Name, humanSize(r.Size), r.Engine)
		return
	}
	var problems []string
	if r.Err != nil {
		problems = append(problems, "dump failed: "+firstLine(r.Err.Error()))
	}
	for _, d := range r.Dests {
		if d.Err != nil {
			problems = append(problems, d.Label+": "+firstLine(d.Err.Error()))
		}
	}
	msg := strings.Join(problems, "; ")
	log.Printf("[%s] backup FAILED: %s", r.Name, msg)
	// An unusable local folder was already announced by RunBackup.
	if r.Err != nil || hasOtherFailure(r) {
		notifier.Notify("Backup failed: "+r.Name, msg)
	}
}

// hasOtherFailure reports whether any destination failed for a reason other
// than being unavailable.
func hasOtherFailure(r *Result) bool {
	for _, d := range r.Dests {
		if d.Err != nil && !errors.Is(d.Err, errDestUnavailable) {
			return true
		}
	}
	return false
}
