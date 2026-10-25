package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/robfig/cron/v3"
)

const (
	engineAuto      = "auto"
	engineMysqldump = "mysqldump"
	engineNative    = "native"

	destFileServer = "fileServer"
	destLocal      = "local"

	defaultRetries = 3
)

type Config struct {
	Databases []*DatabaseConfig `json:"databases"`
}

type MysqlConfig struct {
	Host     string `json:"MYSQL_HOST"`
	Port     int    `json:"MYSQL_PORT"`
	Username string `json:"MYSQL_USERNAME"`
	Password string `json:"MYSQL_PASSWORD"`
	Database string `json:"MYSQL_DATABASE"`
}

type DatabaseConfig struct {
	Name          string        `json:"name"`
	MysqlConfig   MysqlConfig   `json:"mysqlConfig"`
	Engine        string        `json:"engine"`
	MysqldumpPath string        `json:"mysqldumpPath"`
	Cron          string        `json:"cron"`
	Retries       *int          `json:"retries"`
	Destinations  []Destination `json:"destinations"`
}

// Destination is one place a finished backup is delivered. Which fields apply
// depends on Type.
type Destination struct {
	Type string `json:"type"`

	// fileServer
	URL        string `json:"url"`
	Token      string `json:"token"`
	FolderPath string `json:"folderPath"`

	// local
	Path     string `json:"path"`
	KeepLast int    `json:"keepLast"`
}

func (d *Destination) Label() string {
	switch d.Type {
	case destFileServer:
		return "fileServer " + d.URL
	case destLocal:
		return "local " + d.Path
	}
	return d.Type
}

func (d *DatabaseConfig) RetryCount() int {
	if d.Retries == nil {
		return defaultRetries
	}
	return max(*d.Retries, 0)
}

// LoadConfig reads and validates the config file. Every problem found is
// returned together so the user can fix them in one pass.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config %s: %w", path, err)
	}
	cfg := new(Config)
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config %s is not valid JSON: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config %s is invalid:\n%w", path, err)
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	var errs []error
	if len(c.Databases) == 0 {
		return errors.New("  databases: at least one entry is required")
	}
	seen := map[string]bool{}
	for i, db := range c.Databases {
		label := fmt.Sprintf("databases[%d]", i)
		if db == nil {
			errs = append(errs, fmt.Errorf("  %s: empty entry", label))
			continue
		}
		if db.Name != "" {
			label = fmt.Sprintf("%s (%s)", label, db.Name)
		}
		for _, e := range db.validate() {
			errs = append(errs, fmt.Errorf("  %s: %s", label, e))
		}
		if db.Name != "" {
			if seen[db.Name] {
				errs = append(errs, fmt.Errorf("  %s: duplicate name", label))
			}
			seen[db.Name] = true
		}
	}
	return errors.Join(errs...)
}

func (d *DatabaseConfig) validate() []string {
	var p []string
	if d.Name == "" {
		p = append(p, "name is required")
	} else if d.Name != filepath.Base(d.Name) || strings.ContainsAny(d.Name, `/\:*?"<>|`) {
		p = append(p, "name is used in file names and must not contain path or special characters")
	}
	m := d.MysqlConfig
	if m.Host == "" {
		p = append(p, "mysqlConfig.MYSQL_HOST is required")
	}
	if m.Port <= 0 || m.Port > 65535 {
		p = append(p, "mysqlConfig.MYSQL_PORT must be 1-65535")
	}
	if m.Username == "" {
		p = append(p, "mysqlConfig.MYSQL_USERNAME is required")
	}
	if m.Database == "" {
		p = append(p, "mysqlConfig.MYSQL_DATABASE is required")
	}
	switch d.Engine {
	case "":
		d.Engine = engineAuto
	case engineAuto, engineMysqldump, engineNative:
	default:
		p = append(p, fmt.Sprintf("engine %q must be auto, mysqldump or native", d.Engine))
	}
	if d.Cron != "" {
		if _, err := cron.ParseStandard(d.Cron); err != nil {
			p = append(p, fmt.Sprintf("cron %q: %v", d.Cron, err))
		}
	}
	if d.Retries != nil && *d.Retries < 0 {
		p = append(p, "retries must not be negative")
	}
	if len(d.Destinations) == 0 {
		p = append(p, "at least one destination is required")
	}
	for i := range d.Destinations {
		dest := &d.Destinations[i]
		switch dest.Type {
		case destFileServer:
			if dest.URL == "" || dest.Token == "" {
				p = append(p, fmt.Sprintf("destinations[%d]: fileServer needs url and token", i))
			}
		case destLocal:
			if dest.Path == "" {
				p = append(p, fmt.Sprintf("destinations[%d]: local needs path", i))
			}
			if dest.KeepLast < 0 {
				p = append(p, fmt.Sprintf("destinations[%d]: keepLast must not be negative", i))
			}
		default:
			p = append(p, fmt.Sprintf("destinations[%d]: type %q must be fileServer or local", i, dest.Type))
		}
	}
	return p
}
