//go:build !linux

package daemon

import (
	"errors"
	"os"
)

func watchExit(pid int, start string) <-chan struct{} { return pollExit(pid, start) }

func recoverListenerFD(pid, fd int) (*os.File, error) {
	return nil, errors.New("recovering a shared socket from another process needs Linux")
}
