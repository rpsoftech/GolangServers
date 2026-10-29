package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Changes to these paths can alter every component, so they release all of
// them.
var releaseAllPaths = []string{"go.mod", "go.sum", "utility/deploy/"}

// affectedComponents returns the components whose build inputs changed.
// deps maps a component name to the repo-relative, slash-separated directories
// of every package it is built from. Test files and docs cannot change a
// binary, so they are ignored.
func affectedComponents(changed []string, deps map[string][]string) []string {
	var files []string
	for _, f := range changed {
		f = filepath.ToSlash(strings.TrimSpace(f))
		if f == "" || strings.HasSuffix(f, "_test.go") || strings.HasSuffix(f, ".md") {
			continue
		}
		files = append(files, f)
	}

	releaseAll := slices.ContainsFunc(files, func(f string) bool {
		return slices.ContainsFunc(releaseAllPaths, func(p string) bool {
			return f == p || (strings.HasSuffix(p, "/") && strings.HasPrefix(f, p))
		})
	})

	var out []string
	for _, c := range components {
		if releaseAll || slices.ContainsFunc(files, func(f string) bool {
			return slices.ContainsFunc(deps[c.Name], func(dir string) bool {
				return strings.HasPrefix(f, dir+"/")
			})
		}) {
			out = append(out, c.Name)
		}
	}
	return out
}

// componentDeps lists the repo directories each component is built from, for
// every OS it targets because build tags change the import graph.
func componentDeps() (map[string][]string, error) {
	root, err := gitOutput("rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	root = filepath.ToSlash(strings.TrimSpace(root))

	deps := make(map[string][]string, len(components))
	for _, c := range components {
		var oses []string
		for _, t := range c.Targets {
			if !slices.Contains(oses, t.OS) {
				oses = append(oses, t.OS)
			}
		}
		for _, goos := range oses {
			cmd := exec.Command("go", "list", "-deps", "-f", "{{if .Module}}{{if .Module.Main}}{{.Dir}}{{end}}{{end}}", c.Package)
			cmd.Env = append(cmd.Environ(), "GOOS="+goos, "CGO_ENABLED=0")
			out, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("go list %s (%s): %w", c.Package, goos, err)
			}
			for _, dir := range strings.Fields(string(out)) {
				rel, ok := strings.CutPrefix(filepath.ToSlash(dir), root+"/")
				if ok && !slices.Contains(deps[c.Name], rel) {
					deps[c.Name] = append(deps[c.Name], rel)
				}
			}
		}
	}
	return deps, nil
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// affected prints the comma separated components changed between two commits,
// empty when none are. Used by release.yml to pick what to ship to STAGING.
func affected(before, after string) error {
	out, err := gitOutput("diff", "--name-only", before, after)
	if err != nil {
		return err
	}
	deps, err := componentDeps()
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(affectedComponents(strings.Split(out, "\n"), deps), ","))
	return nil
}
