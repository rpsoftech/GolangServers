package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Ecosystem struct {
	Apps []AppConfig `json:"apps"`
}

// LoadEcosystem reads an ecosystem file (JSON, or YAML for .yaml/.yml). A
// relative cwd is resolved against the file's directory; a missing cwd
// defaults to that directory. When env is not empty, the matching env_<env>
// block is merged over env. Warnings are dropped; use LoadEcosystemWarn to
// see them.
func LoadEcosystem(path, env string) ([]AppConfig, error) {
	apps, _, err := LoadEcosystemWarn(path, env)
	return apps, err
}

// LoadEcosystemWarn is LoadEcosystem plus warnings for unknown keys.
func LoadEcosystemWarn(path, env string) ([]AppConfig, []string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	b, err = ecosystemJSON(path, b)
	if err != nil {
		return nil, nil, err
	}
	var eco Ecosystem
	if err := json.Unmarshal(b, &eco); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	warns := unknownKeyWarnings(b)
	apps, err := finishEcosystem(path, env, eco)
	return apps, warns, err
}

func finishEcosystem(path, env string, eco Ecosystem) ([]AppConfig, error) {
	if len(eco.Apps) == 0 {
		return nil, fmt.Errorf("%s: no apps defined", path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(abs)
	for i := range eco.Apps {
		a := &eco.Apps[i]
		if a.Name == "" {
			a.Name = DefaultName(a.Script)
		}
		switch {
		case a.Cwd == "":
			a.Cwd = dir
		case !filepath.IsAbs(a.Cwd):
			a.Cwd = filepath.Join(dir, a.Cwd)
		}
		if env != "" {
			prof, ok := a.EnvProfiles[env]
			if !ok {
				return nil, fmt.Errorf("app %q has no env_%s block", a.Name, env)
			}
			merged := map[string]string{}
			for k, v := range a.Env {
				merged[k] = v
			}
			for k, v := range prof {
				merged[k] = v
			}
			a.Env = merged
		}
	}
	return eco.Apps, nil
}

// DefaultName derives a process name from a script or command line.
func DefaultName(script string) string {
	f := strings.Fields(script)
	if len(f) == 0 {
		return ""
	}
	base := filepath.Base(f[0])
	return strings.TrimSuffix(base, filepath.Ext(base))
}
