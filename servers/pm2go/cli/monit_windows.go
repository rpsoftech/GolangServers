//go:build windows

package cli

import "os"

// notifyResize does nothing on Windows (no SIGWINCH); the dashboard re-reads
// the size on every one-second refresh instead.
func notifyResize(ch chan<- os.Signal) {}
