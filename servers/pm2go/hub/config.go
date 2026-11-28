// Package hub is the web dashboard server: it accepts agent connections,
// stores metrics and events, evaluates alert rules and notifies Telegram.
package hub

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is stored in <home>/config.json. Secrets are generated on first run.
type Config struct {
	Listen        string `json:"listen"`
	AdminPassword string `json:"admin_password"`
	AgentKey      string `json:"agent_key"`
	SessionSecret string `json:"session_secret"`
	RetentionDays int    `json:"retention_days"`
	// Optional built-in TLS; command line flags override these.
	TLSCert    string `json:"tls_cert,omitempty"`
	TLSKey     string `json:"tls_key,omitempty"`
	ACMEDomain string `json:"acme_domain,omitempty"`
	ACMEHTTP   string `json:"acme_http,omitempty"`
}

// Home is $PMGO_HUB_HOME or ~/.pm2go-hub.
func Home() string {
	if h := os.Getenv("PMGO_HUB_HOME"); h != "" {
		return h
	}
	u, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "pm2go-hub")
	}
	return filepath.Join(u, ".pm2go-hub")
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// LoadConfig reads the config, creating it with fresh secrets when absent.
// created reports whether a new file was written.
func LoadConfig(home string) (cfg Config, created bool, err error) {
	path := filepath.Join(home, "config.json")
	b, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(b, &cfg); err != nil {
			return cfg, false, err
		}
	} else if !os.IsNotExist(err) {
		return cfg, false, err
	} else {
		created = true
	}
	dirty := created
	set := func(p *string, v string) {
		if *p == "" {
			*p, dirty = v, true
		}
	}
	set(&cfg.Listen, ":8080")
	set(&cfg.AdminPassword, randHex(8))
	set(&cfg.AgentKey, randHex(16))
	set(&cfg.SessionSecret, randHex(32))
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays, dirty = 7, true
	}
	if dirty {
		if err := os.MkdirAll(home, 0o700); err != nil {
			return cfg, created, err
		}
		out, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(path, out, 0o600); err != nil {
			return cfg, created, err
		}
	}
	return cfg, created, nil
}
