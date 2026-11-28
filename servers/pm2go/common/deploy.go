package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DefaultKeepReleases is how many release directories a deploy keeps.
const DefaultKeepReleases = 5

// DefaultDeployRef is the ref deployed when a deploy block sets none.
const DefaultDeployRef = "origin/main"

// StringList decodes a JSON string or a list of strings.
type StringList []string

func (s *StringList) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*s = StringList{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return fmt.Errorf("expected a string or a list of strings")
	}
	*s = many
	return nil
}

// DeployConfig is one environment of the ecosystem "deploy" block
// (`pm2go deploy <env> ...`). Hook values are shell snippets that run on the
// remote host (pre_deploy_local runs on this machine); every other value is
// data and is always quoted before it reaches a shell.
type DeployConfig struct {
	User           string            `json:"user,omitempty"`
	Host           StringList        `json:"host"`
	Ref            string            `json:"ref,omitempty"`
	Repo           string            `json:"repo,omitempty"`
	Path           string            `json:"path"`
	SSHOptions     []string          `json:"ssh_options,omitempty"`
	PreSetup       string            `json:"pre_setup,omitempty"`
	PostSetup      string            `json:"post_setup,omitempty"`
	PreDeployLocal string            `json:"pre_deploy_local,omitempty"`
	PreDeploy      string            `json:"pre_deploy,omitempty"`
	PostDeploy     string            `json:"post_deploy,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	KeepReleases   int               `json:"keep_releases,omitempty"`
}

var (
	hostRe   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:%\[\]-]*$`)
	userRe   = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)
	envKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// Keep returns keep_releases or the default.
func (d DeployConfig) Keep() int {
	if d.KeepReleases <= 0 {
		return DefaultKeepReleases
	}
	return d.KeepReleases
}

// RefOrDefault returns the ref to deploy.
func (d DeployConfig) RefOrDefault() string {
	if d.Ref == "" {
		return DefaultDeployRef
	}
	return d.Ref
}

// Targets returns the ssh destinations (user@host or host).
func (d DeployConfig) Targets() []string {
	var out []string
	for _, h := range d.Host {
		if d.User != "" {
			out = append(out, d.User+"@"+h)
		} else {
			out = append(out, h)
		}
	}
	return out
}

// Validate checks values that end up in a command line or script. needRepo is
// true for setup and update, the only commands that clone.
func (d DeployConfig) Validate(needRepo bool) error {
	if len(d.Host) == 0 {
		return fmt.Errorf("deploy: host is required")
	}
	for _, h := range d.Host {
		if !hostRe.MatchString(h) {
			return fmt.Errorf("deploy: invalid host %q", h)
		}
	}
	if d.User != "" && !userRe.MatchString(d.User) {
		return fmt.Errorf("deploy: invalid user %q", d.User)
	}
	if strings.TrimSpace(d.Path) == "" || d.Path == "/" {
		return fmt.Errorf("deploy: path is required and must not be /")
	}
	if needRepo && strings.TrimSpace(d.Repo) == "" {
		return fmt.Errorf("deploy: repo is required")
	}
	if strings.HasPrefix(d.Repo, "-") {
		return fmt.Errorf("deploy: repo must not start with '-'")
	}
	if strings.HasPrefix(d.RefOrDefault(), "-") {
		return fmt.Errorf("deploy: ref must not start with '-'")
	}
	if d.KeepReleases < 0 {
		return fmt.Errorf("deploy: keep_releases must not be negative")
	}
	for k := range d.Env {
		if !envKeyRe.MatchString(k) {
			return fmt.Errorf("deploy: invalid env name %q", k)
		}
	}
	for name, v := range map[string]string{
		"path": d.Path, "repo": d.Repo, "ref": d.Ref, "user": d.User,
		"pre_setup": d.PreSetup, "post_setup": d.PostSetup, "pre_deploy_local": d.PreDeployLocal,
		"pre_deploy": d.PreDeploy, "post_deploy": d.PostDeploy,
	} {
		if strings.ContainsRune(v, 0) {
			return fmt.Errorf("deploy: %s contains a NUL byte", name)
		}
	}
	for k, v := range d.Env {
		if strings.ContainsRune(v, 0) {
			return fmt.Errorf("deploy: env %s contains a NUL byte", k)
		}
	}
	for _, o := range d.SSHOptions {
		if strings.ContainsRune(o, 0) {
			return fmt.Errorf("deploy: ssh_options contain a NUL byte")
		}
	}
	return nil
}

// ConvertEcosystem turns an ecosystem file into JSON bytes. The YAML support
// (separate change) replaces this to feed YAML files through the JSON path.
var ConvertEcosystem = func(path string, b []byte) ([]byte, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return nil, fmt.Errorf("%s: YAML ecosystem files are not supported by deploy yet", path)
	}
	return b, nil
}

// LoadDeploy reads the deploy block for env from an ecosystem file.
func LoadDeploy(path, env string) (DeployConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return DeployConfig{}, err
	}
	if b, err = ConvertEcosystem(path, b); err != nil {
		return DeployConfig{}, err
	}
	var eco struct {
		Deploy map[string]DeployConfig `json:"deploy"`
	}
	if err := json.Unmarshal(b, &eco); err != nil {
		return DeployConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg, ok := eco.Deploy[env]
	if !ok {
		return DeployConfig{}, fmt.Errorf("%s: no deploy block for environment %q", path, env)
	}
	return cfg, nil
}
