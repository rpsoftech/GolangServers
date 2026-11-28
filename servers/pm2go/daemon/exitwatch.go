package daemon

import "time"

const exitPollInterval = 500 * time.Millisecond

// pollExit returns a channel that is closed when the pid is gone: not running,
// a zombie, or running with another start time (the pid was reused).
func pollExit(pid int, start string) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		for processAlive(pid) {
			if st, err := procStartTime(pid); err != nil || st != start {
				return
			}
			time.Sleep(exitPollInterval)
		}
	}()
	return ch
}
