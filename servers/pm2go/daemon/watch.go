package daemon

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type fileSig struct {
	mod  int64
	size int64
}

const maxWatchFiles = 50000

var defaultIgnore = []string{".git", "node_modules", ".pm2go"}

// ignored reports whether rel is excluded from watching. Patterns are
// relative to the watched root: "*" stays inside a segment, "**" crosses
// directories, a leading "/" anchors to the root, a trailing "/" matches
// directories only, and a bare name matches at any depth.
func ignored(rel string, isDir bool, extra []string) bool {
	rel = filepath.ToSlash(rel)
	for _, pat := range append(append([]string{}, defaultIgnore...), extra...) {
		if p, ok := compileIgnore(pat); ok && p.matches(rel, isDir) {
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
		if ignored(rel, d.IsDir(), extra) {
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
