package daemon

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	tailInterval = 200 * time.Millisecond
	maxLineBytes = 64 << 10 // longer partial lines are reported without waiting for a newline
)

// logTail follows a log file that a child writes to directly (O_APPEND), so
// the file stays valid when the daemon restarts. It reads new bytes by offset
// and reports complete lines, and it rotates the file by copy-truncate when it
// grows beyond maxSize. Rotated files are compressed as name.1.gz (newest) ...
// name.N.gz.
type logTail struct {
	mu      sync.Mutex
	gzMu    sync.Mutex // serialises shifting and compressing rotated files
	path    string
	maxSize int64 // 0 disables rotation
	retain  int
	onLine  func(line string)
	off     int64
	partial []byte
	closed  bool

	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

// startTail creates the file when missing and follows it from its current
// end, so only lines written from now on are reported.
func startTail(path string, maxSize int64, retain int, onLine func(string)) (*logTail, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0o644)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	f.Close()
	if err != nil {
		return nil, err
	}
	t := &logTail{path: path, maxSize: maxSize, retain: retain, onLine: onLine, off: st.Size(),
		stop: make(chan struct{}), done: make(chan struct{})}
	go t.run()
	return t, nil
}

func (t *logTail) run() {
	defer close(t.done)
	tick := time.NewTicker(tailInterval)
	defer tick.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-tick.C:
			t.sync()
		}
	}
}

// sync reads everything written so far, reports the lines and rotates when
// needed. The supervisor calls it when a child exits so no line is late.
func (t *logTail) sync() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.poll()
}

// poll does the work of sync. Caller holds t.mu.
func (t *logTail) poll() {
	f, err := os.Open(t.path)
	if err != nil {
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return
	}
	if st.Size() < t.off { // truncated by flush or by someone else
		t.off = 0
		t.partial = nil
	}
	buf := make([]byte, 64<<10)
	for {
		n, err := f.ReadAt(buf, t.off)
		if n > 0 {
			t.off += int64(n)
			t.feed(buf[:n])
		}
		if err != nil || n == 0 {
			break
		}
	}
	if t.maxSize > 0 && t.off > t.maxSize {
		t.rotate(f)
	}
}

// feed splits data into lines. Caller holds t.mu.
func (t *logTail) feed(b []byte) {
	if t.onLine == nil {
		return
	}
	data := append(t.partial, b...)
	for {
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}
		t.onLine(string(bytes.TrimRight(data[:i], "\r")))
		data = data[i+1:]
	}
	if len(data) > maxLineBytes {
		t.onLine(string(data))
		data = nil
	}
	t.partial = append([]byte(nil), data...)
}

// rotate copies the bytes read so far to a temporary file, truncates the log
// and compresses the copy in the background. The child keeps its descriptor;
// O_APPEND puts its next write at the new end. Bytes written between the copy
// and the truncate are lost. Caller holds t.mu; f is the open log file.
func (t *logTail) rotate(f *os.File) {
	pending := fmt.Sprintf("%s.%d.rot", t.path, time.Now().UnixNano())
	out, err := os.OpenFile(pending, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return
	}
	_, cerr := io.Copy(out, io.NewSectionReader(f, 0, t.off))
	if oerr := out.Close(); cerr != nil || oerr != nil {
		os.Remove(pending)
		return
	}
	if err := os.Truncate(t.path, 0); err != nil {
		os.Remove(pending)
		return
	}
	t.off = 0
	go func() {
		t.gzMu.Lock()
		defer t.gzMu.Unlock()
		// Shift name.N-1.gz -> name.N.gz; the oldest falls off.
		_ = os.Remove(fmt.Sprintf("%s.%d.gz", t.path, t.retain))
		for i := t.retain - 1; i >= 1; i-- {
			_ = os.Rename(fmt.Sprintf("%s.%d.gz", t.path, i), fmt.Sprintf("%s.%d.gz", t.path, i+1))
		}
		gzipFile(pending, t.path+".1.gz")
	}()
}

// gzipFile compresses src into dst, then removes src.
func gzipFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return
	}
	zw := gzip.NewWriter(out)
	_, cerr := io.Copy(zw, in)
	zerr := zw.Close()
	oerr := out.Close()
	if cerr != nil || zerr != nil || oerr != nil {
		os.Remove(tmp)
		return
	}
	if os.Rename(tmp, dst) == nil {
		os.Remove(src)
	}
}

// Truncate empties the file.
func (t *logTail) Truncate() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := os.Truncate(t.path, 0); err != nil && !os.IsNotExist(err) {
		return err
	}
	t.off = 0
	t.partial = nil
	return nil
}

// Close stops the tailer after a last read and reports a trailing partial line.
func (t *logTail) Close() {
	t.stopOnce.Do(func() { close(t.stop) })
	<-t.done
	t.mu.Lock()
	defer t.mu.Unlock()
	t.poll()
	t.closed = true
	if len(t.partial) > 0 && t.onLine != nil {
		t.onLine(string(t.partial))
		t.partial = nil
	}
}
