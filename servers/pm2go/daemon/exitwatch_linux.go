package daemon

import (
	"os"

	"golang.org/x/sys/unix"
)

// watchExit returns a channel closed when the process is gone. It waits on a
// pidfd (Linux 5.3+), which becomes readable at exit, and falls back to
// polling /proc when pidfd_open is not available.
func watchExit(pid int, start string) <-chan struct{} {
	fd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return pollExit(pid, start)
	}
	// Reject a pid that was reused between the check and the open.
	if st, err := procStartTime(pid); err != nil || st != start {
		unix.Close(fd)
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		defer unix.Close(fd)
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		for {
			n, err := unix.Poll(fds, -1)
			if err == unix.EINTR {
				continue
			}
			if err != nil || n > 0 {
				return
			}
		}
	}()
	return ch
}

// recoverListenerFD duplicates descriptor fd of another process with
// pidfd_getfd (Linux 5.6+, same user), so the daemon can hold a shared socket
// again after it adopted the children that kept it open.
func recoverListenerFD(pid, fd int) (*os.File, error) {
	pfd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(pfd)
	nfd, err := unix.PidfdGetfd(pfd, fd, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(nfd), "shared-listener"), nil
}
