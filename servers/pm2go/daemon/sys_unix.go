//go:build !windows

package daemon

import (
	"os/exec"
	"syscall"
)

// Each child runs in its own process group so signals reach its descendants.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalGroup(pid int, kill bool) {
	sig := syscall.SIGTERM
	if kill {
		sig = syscall.SIGKILL
	}
	if err := syscall.Kill(-pid, sig); err != nil {
		_ = syscall.Kill(pid, sig)
	}
}

func DetachAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }
