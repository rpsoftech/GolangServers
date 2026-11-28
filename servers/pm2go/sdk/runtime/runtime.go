// Package runtime lets an application started by pm2go use the shared
// listening socket and the readiness protocol. It depends on the standard
// library only.
//
// Under pm2go the daemon opens the listener once per app and passes it to
// every instance as file descriptor 3 (systemd socket activation style,
// LISTEN_FDS=1). Outside pm2go the helpers fall back to a normal listener, so
// the same binary runs both ways.
package runtime

import (
	"errors"
	"net"
	"os"
	"strconv"
	"sync"
)

const (
	envListenFDs  = "LISTEN_FDS"
	envListenPID  = "LISTEN_PID"
	envReadyFD    = "PMGO_READY_FD"
	envPort       = "PORT"
	listenFDStart = 3
	defaultAddr   = ":8080"
)

// Inherited returns the listener passed by pm2go. The bool is false when the
// process was not started with a shared socket.
func Inherited() (net.Listener, bool, error) {
	inhMu.Lock()
	defer inhMu.Unlock()
	if inhDone {
		return inhLn, inhLn != nil, inhErr
	}
	inhDone = true
	inhLn, inhErr = inherit()
	return inhLn, inhLn != nil, inhErr
}

var (
	inhMu   sync.Mutex
	inhDone bool
	inhLn   net.Listener
	inhErr  error
)

func inherit() (net.Listener, error) {
	n, _ := strconv.Atoi(os.Getenv(envListenFDs))
	if n < 1 {
		return nil, nil
	}
	if pid := os.Getenv(envListenPID); pid != "" && pid != strconv.Itoa(os.Getpid()) {
		return nil, nil // meant for another process
	}
	return inheritedFD(listenFDStart)
}

// ListenerOn returns the inherited listener, or net.Listen(network, addr)
// when the process is not under pm2go (or has no shared socket).
func ListenerOn(network, addr string) (net.Listener, error) {
	ln, ok, err := Inherited()
	if err != nil || ok {
		return ln, err
	}
	return net.Listen(network, addr)
}

// Listener is ListenerOn with a tcp fallback on ":$PORT", or ":8080" when
// PORT is not set.
func Listener() (net.Listener, error) {
	addr := defaultAddr
	if p := os.Getenv(envPort); p != "" {
		addr = ":" + p
	}
	return ListenerOn("tcp", addr)
}

var readyOnce sync.Once

// Ready tells pm2go that the application is ready to serve. Call it after the
// listener is open and the server is initialised. It does nothing when the app
// is not under pm2go or wait_ready is off, and is safe to call more than once.
func Ready() error {
	fd, err := strconv.Atoi(os.Getenv(envReadyFD))
	if err != nil || fd < 3 {
		return nil
	}
	var werr error
	readyOnce.Do(func() {
		f := os.NewFile(uintptr(fd), "pmgo-ready")
		if f == nil {
			werr = errors.New("pm2go: invalid readiness descriptor")
			return
		}
		defer f.Close()
		_, werr = f.WriteString("ready\n")
	})
	return werr
}
