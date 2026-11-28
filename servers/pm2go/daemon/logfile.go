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

// RotatingFile is an append-only log file that rotates by size. Rotated
// files are compressed as name.1.gz (newest) ... name.N.gz.
type RotatingFile struct {
	mu      sync.Mutex
	path    string
	maxSize int64 // 0 disables rotation
	retain  int
	f       *os.File
	size    int64
	onLine  func(line string)
	partial []byte
}

func openRotating(path string, maxSize int64, retain int, onLine func(string)) (*RotatingFile, error) {
	r := &RotatingFile{path: path, maxSize: maxSize, retain: retain, onLine: onLine}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := r.open(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RotatingFile) open() error {
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	r.f, r.size = f, st.Size()
	return nil
}

// Write implements io.Writer. It writes raw bytes to the file and calls onLine
// once per complete line.
func (r *RotatingFile) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return 0, os.ErrClosed
	}
	if r.maxSize > 0 && r.size+int64(len(b)) > r.maxSize && r.size > 0 {
		r.rotate()
	}
	n, err := r.f.Write(b)
	r.size += int64(n)
	if r.onLine != nil {
		data := append(r.partial, b...)
		for {
			i := bytes.IndexByte(data, '\n')
			if i < 0 {
				break
			}
			r.onLine(string(bytes.TrimRight(data[:i], "\r")))
			data = data[i+1:]
		}
		if len(data) > 64<<10 { // cap runaway lines
			r.onLine(string(data))
			data = nil
		}
		r.partial = append([]byte(nil), data...)
	}
	return n, err
}

func (r *RotatingFile) rotate() {
	r.f.Close()
	// Shift name.N-1.gz -> name.N.gz; the oldest falls off.
	_ = os.Remove(fmt.Sprintf("%s.%d.gz", r.path, r.retain))
	for i := r.retain - 1; i >= 1; i-- {
		_ = os.Rename(fmt.Sprintf("%s.%d.gz", r.path, i), fmt.Sprintf("%s.%d.gz", r.path, i+1))
	}
	pending := fmt.Sprintf("%s.%d.rot", r.path, time.Now().UnixNano())
	if err := os.Rename(r.path, pending); err == nil {
		go gzipFile(pending, r.path+".1.gz")
	}
	if err := r.open(); err != nil {
		r.f = nil
	}
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

// Truncate empties the current file.
func (r *RotatingFile) Truncate() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return os.Truncate(r.path, 0)
	}
	if err := r.f.Truncate(0); err != nil {
		return err
	}
	r.size = 0
	return nil
}

func (r *RotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	if len(r.partial) > 0 && r.onLine != nil {
		r.onLine(string(r.partial))
		r.partial = nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}
