//go:build !windows

package cli

import (
	"os"
	"os/signal"
	"syscall"
)

// notifyResize sends on ch when the terminal is resized.
func notifyResize(ch chan<- os.Signal) { signal.Notify(ch, syscall.SIGWINCH) }
