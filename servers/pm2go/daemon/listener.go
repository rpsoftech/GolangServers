package daemon

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// sharedListener is a listening socket owned by the daemon. Only the
// duplicated descriptor (file) is kept open; every instance receives it as an
// inherited descriptor, so the socket outlives any single child.
type sharedListener struct {
	network, addr string
	file          *os.File
}

func (s *sharedListener) close() {
	_ = s.file.Close()
	if s.network == "unix" {
		_ = os.Remove(s.addr)
	}
}

// listeners is the registry of shared sockets, one per app name.
type listeners struct {
	mu sync.Mutex
	m  map[string]*sharedListener
}

func newListeners() *listeners { return &listeners{m: map[string]*sharedListener{}} }

func openShared(network, addr string) (*sharedListener, error) {
	if network == "unix" {
		clearStaleSocket(addr)
	}
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	fl, ok := ln.(interface{ File() (*os.File, error) })
	if !ok {
		_ = ln.Close()
		return nil, fmt.Errorf("listen network %q cannot be shared", network)
	}
	if ul, ok := ln.(*net.UnixListener); ok {
		ul.SetUnlinkOnClose(false) // the shared descriptor keeps serving
	}
	f, err := fl.File()
	_ = ln.Close()
	if err != nil {
		return nil, err
	}
	return &sharedListener{network: network, addr: addr, file: f}, nil
}

// clearStaleSocket removes a unix socket file that nothing listens on.
func clearStaleSocket(path string) {
	fi, err := os.Lstat(path)
	if err != nil || fi.Mode()&os.ModeSocket == 0 {
		return
	}
	if c, err := net.Dial("unix", path); err == nil {
		_ = c.Close()
		return
	}
	_ = os.Remove(path)
}

// acquire returns the shared socket for the app, opening it on first use.
// A changed address replaces the socket; running children keep the old one.
func (l *listeners) acquire(cfg common.AppConfig) (*sharedListener, error) {
	network, addr := cfg.ListenSpec()
	if addr == "" {
		return nil, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if cur := l.m[cfg.Name]; cur != nil {
		if cur.network == network && cur.addr == addr {
			return cur, nil
		}
		cur.close()
		delete(l.m, cfg.Name)
	}
	s, err := openShared(network, addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s %s: %w", network, addr, err)
	}
	l.m[cfg.Name] = s
	return s, nil
}

// release closes the shared socket of an app.
func (l *listeners) release(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if s := l.m[name]; s != nil {
		s.close()
		delete(l.m, name)
	}
}

func (l *listeners) closeAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for n, s := range l.m {
		s.close()
		delete(l.m, n)
	}
}

// adopt registers the shared socket of an app again, taking it from an adopted
// child that inherited it as descriptor 3. Without it the daemon could not
// hand the socket to later children. Failure is not fatal: the app keeps
// running, a restart then has to bind the address itself.
func (l *listeners) adopt(cfg common.AppConfig, pid int) {
	network, addr := cfg.ListenSpec()
	if addr == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.m[cfg.Name] != nil {
		return
	}
	f, err := recoverListenerFD(pid, 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pm2go: %q: cannot recover the shared socket from pid %d: %v\n", cfg.Name, pid, err)
		return
	}
	if st, err := f.Stat(); err != nil || st.Mode()&os.ModeSocket == 0 {
		f.Close()
		return
	}
	l.m[cfg.Name] = &sharedListener{network: network, addr: addr, file: f}
}
