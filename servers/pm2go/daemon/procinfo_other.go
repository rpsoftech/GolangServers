//go:build !linux

package daemon

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Non-Linux builds only exist so the code can be developed on a laptop; they
// ask ps instead of reading /proc.

func psField(pid int, field string) (string, error) {
	out, err := exec.Command("ps", "-o", field+"=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// procStartTime returns the process start time as ps prints it (one second
// resolution). A reused pid has a different value.
func procStartTime(pid int) (string, error) {
	s, err := psField(pid, "lstart")
	if err == nil && s == "" {
		err = syscall.ESRCH
	}
	return s, err
}

func procCmdline(pid int) (string, error) { return psField(pid, "command") }

func processAlive(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil && err != syscall.EPERM {
		return false
	}
	st, err := psField(pid, "stat")
	return err == nil && st != "" && !strings.HasPrefix(st, "Z")
}
