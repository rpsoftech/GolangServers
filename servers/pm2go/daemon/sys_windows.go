//go:build windows

package daemon

import (
	"os"
	"os/exec"
	"syscall"
)

// Windows is a development target only: supervision compiles and starts
// children, but there are no process groups and no graceful signal, so stop
// kills the process itself. Production is Linux.

func setProcAttr(cmd *exec.Cmd) {}

func signalGroup(pid int, kill bool) {
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}

// DetachAttr starts the daemon detached from the console
// (CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS).
func DetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x00000008}
}
