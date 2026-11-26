package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/rpsoftech/golang-servers/utility/updater"
)

// skipUpdateEnv is set for the process started after an update, so it does not
// check again.
const skipUpdateEnv = "MYSQL_BACKUP_SKIP_UPDATE"

// shouldCheckUpdate: only published builds update, and never the process that
// was just started by an update.
func shouldCheckUpdate(published bool, skipEnv string) bool {
	return published && skipEnv == ""
}

// restartArgs are the arguments for the updated process. The parent keeps the
// console and does the "Press Enter" pause, so the child must not.
func restartArgs(args []string) []string {
	return append([]string{"--no-pause"}, args...)
}

// runChild runs exe with this process's input and output and returns its exit
// code.
func runChild(exe string, args []string) int {
	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), skipUpdateEnv+"=1")
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "ERROR: cannot start the updated version:", err)
		return 1
	}
	return 0
}

// updateAndRestart checks for a newer signed release before a quick run. When
// it installs one it starts the new executable with the same arguments, waits
// for it and returns its exit code with true. A failed check is only logged:
// the backup then runs with the current version.
func updateAndRestart() (int, bool) {
	if !shouldCheckUpdate(updater.Published(), os.Getenv(skipUpdateEnv)) {
		return 0, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	updated, err := updater.SelfUpdate(ctx, updater.Config{
		Component:      updater.MysqlBackupExeProjectName,
		CurrentVersion: updater.ParseVersion(version),
	})
	if err != nil {
		log.Printf("[updater] %v (continuing with the current version)", err)
		return 0, false
	}
	if !updated {
		return 0, false
	}
	log.Println("[updater] update installed, restarting")
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: updated, but cannot find the new executable:", err)
		return 1, true
	}
	return runChild(exe, restartArgs(os.Args[1:])), true
}
