package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

const (
	engineAuto      = "auto"
	engineMysqldump = "mysqldump"
	engineNative    = "native"

	destFileServer = "fileServer"
	destLocal      = "local"
	destR2         = "r2"

	defaultRetries = 3
	defaultTimeout = 6 * time.Hour
)

type Config struct {
	// Telegram, when set, sends every backup result to a chat.
	Telegram *TelegramConfig `json:"telegram"`
	// Parallel is how many databases are backed up at the same time (1 to 16).
	// 0 or unset keeps the old behaviour: a quick run goes one database after
	// another, and the daemon puts no limit on schedules that fire together.
	// It speeds up backups over a remote link, where each dump mostly waits on
	// the network; every running backup is one more connection to the server.
	Parallel  int               `json:"parallel"`
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
	Name          string      `json:"name"`
	MysqlConfig   MysqlConfig `json:"mysqlConfig"`
	Engine        string      `json:"engine"`
	MysqldumpPath string      `json:"mysqldumpPath"`
	// StripDefiner removes the DEFINER clause from views, routines, triggers
	// and events, so the dump restores on a server that lacks the original
	// user. Off by default: restored objects then belong to the user running
	// the restore, and SQL SECURITY DEFINER objects run with that user's
	// privileges.
	StripDefiner bool `json:"stripDefiner"`
	// Compress asks MySQL to compress the traffic between the server and this
	// tool. It speeds up backups over a remote link and does nothing useful
	// when the tool runs on the database server. Off by default.
	Compress bool   `json:"compress"`
	Cron     string `json:"cron"`
	// RunOnStart also backs the database up once when the daemon starts.
	RunOnStart bool `json:"runOnStart"`
	Retries    *int `json:"retries"`
	// TimeoutMinutes caps one whole backup (dump and every destination), so a
	// hung database or file server cannot hold the cron job forever. 0 means
	// the default of 360.
	TimeoutMinutes int `json:"timeoutMinutes"`
	// EncryptTo lists age public keys (age1...). When set, the backup is
	// encrypted once and every destination receives the .sql.gz.age file;
	// any one of the matching secret keys can open it.
	EncryptTo    []string      `json:"encryptTo"`
	Destinations []Destination `json:"destinations"`
}

// Destination is one place a finished backup is delivered. Which fields apply
// depends on Type.
type Destination struct {
	Type string `json:"type"`

	// fileServer
	URL        string `json:"url"`
	Token      string `json:"token"`
	FolderPath string `json:"folderPath"`

	// r2 (Cloudflare R2, S3-compatible). Give AccountID, or Endpoint to use
	// another S3 endpoint. FolderPath is the key prefix.
	AccountID       string `json:"accountId"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`

	// local and r2; KeepLast and KeepDays of 0 disable that rule
	Path     string `json:"path"`
	KeepLast int    `json:"keepLast"`
	KeepDays int    `json:"keepDays"`
}

func (d *Destination) Label() string {
	switch d.Type {
	case destFileServer:
		return "fileServer " + d.URL
	case destLocal:
		return "local " + d.Path
	case destR2:
		return "r2 " + path.Join(d.Bucket, d.FolderPath)
	}
	return d.Type
}

func (d *Destination) validateR2(i int) []string {
	var p []string
	bad := func(format string, a ...any) {
		p = append(p, fmt.Sprintf("destinations[%d]: ", i)+fmt.Sprintf(format, a...))
	}
	if d.AccountID == "" && d.Endpoint == "" {
		bad("r2 needs accountId (or endpoint)")
	}
	if d.Bucket == "" || d.AccessKeyID == "" || d.SecretAccessKey == "" {
		bad("r2 needs bucket, accessKeyId and secretAccessKey")
	}
	for _, v := range []string{d.AccountID, d.Endpoint, d.Bucket, d.AccessKeyID, d.SecretAccessKey} {
		if strings.ContainsAny(v, "<>") {
			bad("r2 still holds a placeholder value (%q)", v)
			break
		}
	}
	if d.Endpoint != "" {
		if u, err := url.Parse(d.Endpoint); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			bad("r2 endpoint %q must be a full http(s) URL", d.Endpoint)
		}
	}
	if d.KeepLast < 0 {
		bad("keepLast must not be negative")
	}
	if d.KeepDays < 0 {
		bad("keepDays must not be negative")
	}
	return p
}

func (d *DatabaseConfig) RetryCount() int {
	if d.Retries == nil {
		return defaultRetries
	}
	return max(*d.Retries, 0)
}

func (d *DatabaseConfig) Timeout() time.Duration {
	if d.TimeoutMinutes <= 0 {
		return defaultTimeout
	}
	return time.Duration(d.TimeoutMinutes) * time.Minute
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
	cfg.trimSpaces()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config %s is invalid:\n%w", path, err)
	}
	return cfg, nil
}

// trimSpaces removes stray spaces around host names, user names, URLs and
// similar values, which are easy to leave in when copying (a trailing space in
// a host made the lookup fail). Passwords and tokens are left exactly as given.
func (c *Config) trimSpaces() {
	for _, db := range c.Databases {
		if db == nil {
			continue
		}
		for _, s := range []*string{
			&db.Name, &db.MysqlConfig.Host, &db.MysqlConfig.Username,
			&db.MysqlConfig.Database, &db.Engine, &db.MysqldumpPath, &db.Cron,
		} {
			*s = strings.TrimSpace(*s)
		}
		for i := range db.Destinations {
			d := &db.Destinations[i]
			for _, s := range []*string{
				&d.Type, &d.URL, &d.FolderPath, &d.AccountID, &d.Endpoint,
				&d.Bucket, &d.Path,
			} {
				*s = strings.TrimSpace(*s)
			}
		}
	}
}

func (c *Config) Validate() error {
	var errs []error
	if c.Telegram != nil {
		for _, e := range c.Telegram.validate() {
			errs = append(errs, fmt.Errorf("  %s", e))
		}
	}
	if c.Parallel < 0 || c.Parallel > maxParallel {
		errs = append(errs, fmt.Errorf("  parallel: must be between 1 and %d (or 0 to leave it unset)", maxParallel))
	}
	if len(c.Databases) == 0 {
		errs = append(errs, errors.New("  databases: at least one entry is required"))
		return errors.Join(errs...)
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
	} else if d.RunOnStart {
		p = append(p, "runOnStart needs cron (daemon mode only runs scheduled databases)")
	}
	if len(d.EncryptTo) > 0 {
		if _, err := parseRecipients(d.EncryptTo); err != nil {
			p = append(p, err.Error())
		}
	}
	if d.Retries != nil && *d.Retries < 0 {
		p = append(p, "retries must not be negative")
	}
	if d.TimeoutMinutes < 0 {
		p = append(p, "timeoutMinutes must not be negative")
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
			if dest.KeepDays < 0 {
				p = append(p, fmt.Sprintf("destinations[%d]: keepDays must not be negative", i))
			}
		case destR2:
			p = append(p, dest.validateR2(i)...)
		default:
			p = append(p, fmt.Sprintf("destinations[%d]: type %q must be fileServer, local or r2", i, dest.Type))
		}
	}
	return p
}
