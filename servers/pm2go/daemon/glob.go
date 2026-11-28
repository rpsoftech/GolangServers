package daemon

import (
	"path"
	"strings"
)

// ignorePattern is one compiled ignore_watch pattern.
type ignorePattern struct {
	segs     []string // pattern split on "/"
	dirOnly  bool     // trailing "/": matches directories only
	anyDepth bool     // bare name: matches a path segment at any depth
}

func compileIgnore(pat string) (ignorePattern, bool) {
	pat = strings.TrimSpace(strings.ReplaceAll(pat, "\\", "/"))
	var p ignorePattern
	if strings.HasSuffix(pat, "/") {
		p.dirOnly = true
		pat = strings.TrimRight(pat, "/")
	}
	anchored := strings.HasPrefix(pat, "/")
	pat = strings.TrimLeft(pat, "/")
	if pat == "" {
		return p, false
	}
	p.segs = strings.Split(pat, "/")
	p.anyDepth = !anchored && len(p.segs) == 1
	return p, true
}

// matchSegs matches pattern segments against path segments; "**" matches zero
// or more whole segments.
func matchSegs(pat, parts []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			for len(pat) > 1 && pat[1] == "**" {
				pat = pat[1:]
			}
			if len(pat) == 1 {
				return true
			}
			for i := 0; i <= len(parts); i++ {
				if matchSegs(pat[1:], parts[i:]) {
					return true
				}
			}
			return false
		}
		if len(parts) == 0 {
			return false
		}
		if ok, _ := path.Match(pat[0], parts[0]); !ok {
			return false
		}
		pat, parts = pat[1:], parts[1:]
	}
	return len(parts) == 0
}

// matches reports whether the pattern ignores rel (slash separated, relative
// to the watched root). A pattern that matches a parent directory ignores
// everything below it.
func (p ignorePattern) matches(rel string, isDir bool) bool {
	parts := strings.Split(rel, "/")
	for n := 1; n <= len(parts); n++ {
		dir := n < len(parts) || isDir
		if p.dirOnly && !dir {
			continue
		}
		if p.anyDepth {
			if ok, _ := path.Match(p.segs[0], parts[n-1]); ok {
				return true
			}
			continue
		}
		if matchSegs(p.segs, parts[:n]) {
			return true
		}
	}
	return false
}
