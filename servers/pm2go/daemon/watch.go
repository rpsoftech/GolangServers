package daemon

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type fileSig struct {
	mod  int64
	size int64
}

const maxWatchFiles = 50000

var defaultIgnore = []string{".git", "node_modules", ".pm2go"}

func ignored(rel string, extra []string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, pat := range append(append([]string{}, defaultIgnore...), extra...) {
		for _, part := range parts {
			if ok, _ := filepath.Match(pat, part); ok {
				return true
			}
		}
		if ok, _ := filepath.Match(pat, rel); ok {
			return true
		}
	}
	return false
}

func scanDir(root string, extra []string) map[string]fileSig {
	out := map[string]fileSig{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || rel == "." {
			return nil
		}
		if ignored(rel, extra) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if len(out) >= maxWatchFiles {
			return filepath.SkipAll
		}
		if info, err := d.Info(); err == nil {
			out[rel] = fileSig{info.ModTime().UnixNano(), info.Size()}
		}
		return nil
	})
	return out
}

func changed(a, b map[string]fileSig) bool {
	if len(a) != len(b) {
		return true
	}
	for k, v := range a {
		if b[k] != v {
			return true
		}
	}
	return false
}

// watchLoop polls the cwd and requests a soft restart on change.
func (p *Proc) watchLoop(ctx context.Context) {
	p.mu.Lock()
	root, extra := p.cfg.Cwd, p.cfg.IgnoreWatch
	delay := time.Duration(p.cfg.WatchDelayMs) * time.Millisecond
	p.mu.Unlock()
	if root == "" {
		root, _ = os.Getwd()
	}
	if delay <= 0 {
		delay = time.Second
	}
	snap := scanDir(root, extra)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		cur := scanDir(root, extra)
		if !changed(snap, cur) {
			continue
		}
		select { // debounce: let a burst of writes settle
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		snap = scanDir(root, extra)
		p.requestRestart("watch")
	}
}
