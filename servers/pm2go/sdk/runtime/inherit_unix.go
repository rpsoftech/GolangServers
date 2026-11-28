//go:build unix

package runtime

import (
	"net"
	"os"
	"syscall"
)

func inheritedFD(fd int) (net.Listener, error) {
	syscall.CloseOnExec(fd) // do not leak it into the app's own children
	f := os.NewFile(uintptr(fd), "pmgo-listener")
	defer f.Close() // FileListener dups the descriptor
	return net.FileListener(f)
}
