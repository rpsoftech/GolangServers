//go:build !windows

package daemon

import "os/exec"

func shellCommand(line string) *exec.Cmd { return exec.Command("sh", "-c", line) }
