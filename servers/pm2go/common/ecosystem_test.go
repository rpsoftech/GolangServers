package common

import (
	"os"
	"path/filepath"
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
