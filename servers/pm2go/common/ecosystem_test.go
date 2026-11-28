package common

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeEco(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "eco.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadEcosystemEnvProfile(t *testing.T) {
	p := writeEco(t, `{"apps":[{"script":"./srv","env":{"A":"1","B":"1"},"env_production":{"B":"2"},"cwd":"sub"}]}`)
	apps, err := LoadEcosystem(p, "production")
	if err != nil {
		t.Fatal(err)
	}
	a := apps[0]
	if a.Name != "srv" {
		t.Errorf("name = %q", a.Name)
	}
	if a.Env["A"] != "1" || a.Env["B"] != "2" {
		t.Errorf("env = %v", a.Env)
	}
	if !filepath.IsAbs(a.Cwd) || filepath.Base(a.Cwd) != "sub" {
		t.Errorf("cwd = %q", a.Cwd)
	}
}

func TestLoadEcosystemMissingProfile(t *testing.T) {
	p := writeEco(t, `{"apps":[{"name":"x","script":"s"}]}`)
	if _, err := LoadEcosystem(p, "production"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadEcosystemEmpty(t *testing.T) {
	if _, err := LoadEcosystem(writeEco(t, `{"apps":[]}`), ""); err == nil {
		t.Fatal("expected error")
	}
}

func writeEcoExt(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const ecoJSON = `{"apps":[{"name":"api","script":"./srv","args":["-p","80"],"instances":2,"cwd":"sub",
 "env":{"A":"1","PORT":"3000"},"env_production":{"A":"2"},"ignore_watch":["**/*.log","tmp/"],"max_restarts":5}]}`

const ecoYAML = `
apps:
  - name: api
    script: ./srv
    args: ["-p", "80"]
    instances: 2
    cwd: sub
    env:
      A: "1"
      PORT: 3000
    env_production:
      A: "2"
    ignore_watch:
      - "**/*.log"
      - tmp/
    max_restarts: 5
`

func TestYAMLEquivalentToJSON(t *testing.T) {
	for _, ext := range []string{"yaml", "yml"} {
		j, err := LoadEcosystem(writeEcoExt(t, "e.json", ecoJSON), "")
		if err != nil {
			t.Fatal(err)
		}
		y, err := LoadEcosystem(writeEcoExt(t, "e."+ext, ecoYAML), "")
		if err != nil {
			t.Fatal(err)
		}
		// cwd differs by temp dir; compare relative part.
		j[0].Cwd, y[0].Cwd = filepath.Base(j[0].Cwd), filepath.Base(y[0].Cwd)
		if !reflect.DeepEqual(j, y) {
			t.Errorf("%s: json %+v != yaml %+v", ext, j, y)
		}
	}
}

func TestYAMLEnvProfile(t *testing.T) {
	apps, err := LoadEcosystem(writeEcoExt(t, "e.yaml", ecoYAML), "production")
	if err != nil {
		t.Fatal(err)
	}
	if apps[0].Env["A"] != "2" || apps[0].Env["PORT"] != "3000" {
		t.Errorf("env = %v", apps[0].Env)
	}
	if _, err := LoadEcosystem(writeEcoExt(t, "e.yaml", ecoYAML), "staging"); err == nil {
		t.Error("expected missing profile error")
	}
}

func TestYAMLInvalid(t *testing.T) {
	p := writeEcoExt(t, "bad.yaml", "apps:\n  - name: a\n   script: x\n")
	_, err := LoadEcosystem(p, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad.yaml") || !strings.Contains(err.Error(), "line") {
		t.Errorf("error lacks file name or line: %v", err)
	}
}

func TestUnknownExtension(t *testing.T) {
	if _, err := LoadEcosystem(writeEcoExt(t, "eco.conf", ecoJSON), ""); err != nil {
		t.Errorf("json content: %v", err)
	}
	if _, err := LoadEcosystem(writeEcoExt(t, "eco.txt", ecoYAML), ""); err != nil {
		t.Errorf("yaml content: %v", err)
	}
	_, err := LoadEcosystem(writeEcoExt(t, "eco.txt", "apps: [\n"), "")
	if err == nil || !strings.Contains(err.Error(), "eco.txt") {
		t.Errorf("want clear error, got %v", err)
	}
}

func TestUnknownKeyWarnings(t *testing.T) {
	body := "apps:\n  - name: api\n    script: x\n    max_restart: 3\n    env_dev: {A: b}\nextra: 1\n"
	for _, name := range []string{"e.yaml", "e.json"} {
		content := body
		if name == "e.json" {
			content = `{"apps":[{"name":"api","script":"x","max_restart":3,"env_dev":{"A":"b"}}],"extra":1}`
		}
		apps, warns, err := LoadEcosystemWarn(writeEcoExt(t, name, content), "")
		if err != nil || len(apps) != 1 {
			t.Fatalf("%s: apps=%v err=%v", name, apps, err)
		}
		if len(warns) != 2 || !strings.Contains(warns[0], `"extra"`) || !strings.Contains(warns[1], `app "api"`) || !strings.Contains(warns[1], "max_restart") {
			t.Errorf("%s: warnings = %q", name, warns)
		}
	}
}

func TestIsEcosystemFile(t *testing.T) {
	for f, want := range map[string]bool{"a.json": true, "a.YAML": true, "a.yml": true, "./srv": false, "node app.js": false, "a.js": false} {
		if IsEcosystemFile(f) != want {
			t.Errorf("%q: want %v", f, want)
		}
	}
}
