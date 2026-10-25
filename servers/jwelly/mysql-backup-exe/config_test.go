package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validDB() *DatabaseConfig {
	return &DatabaseConfig{
		Name:        "shop",
		MysqlConfig: MysqlConfig{Host: "127.0.0.1", Port: 3306, Username: "root", Password: "x", Database: "shop"},
		Destinations: []Destination{
			{Type: destLocal, Path: "backups"},
		},
	}
}

func TestValidateAcceptsMinimalConfig(t *testing.T) {
	db := validDB()
	cfg := &Config{Databases: []*DatabaseConfig{db}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db.Engine != engineAuto {
		t.Errorf("engine default = %q, want %q", db.Engine, engineAuto)
	}
	if got := db.RetryCount(); got != defaultRetries {
		t.Errorf("RetryCount default = %d, want %d", got, defaultRetries)
	}
}

func TestValidateRejectsProblems(t *testing.T) {
	neg := -1
	cases := []struct {
		name   string
		mutate func(*DatabaseConfig)
		want   string
	}{
		{"missing name", func(d *DatabaseConfig) { d.Name = "" }, "name is required"},
		{"path in name", func(d *DatabaseConfig) { d.Name = "a/b" }, "must not contain path"},
		{"special char in name", func(d *DatabaseConfig) { d.Name = "a:b" }, "must not contain path"},
		{"missing host", func(d *DatabaseConfig) { d.MysqlConfig.Host = "" }, "MYSQL_HOST"},
		{"port zero", func(d *DatabaseConfig) { d.MysqlConfig.Port = 0 }, "MYSQL_PORT"},
		{"port too big", func(d *DatabaseConfig) { d.MysqlConfig.Port = 70000 }, "MYSQL_PORT"},
		{"missing user", func(d *DatabaseConfig) { d.MysqlConfig.Username = "" }, "MYSQL_USERNAME"},
		{"missing database", func(d *DatabaseConfig) { d.MysqlConfig.Database = "" }, "MYSQL_DATABASE"},
		{"bad engine", func(d *DatabaseConfig) { d.Engine = "pg_dump" }, "engine"},
		{"bad cron", func(d *DatabaseConfig) { d.Cron = "not a cron" }, "cron"},
		{"negative retries", func(d *DatabaseConfig) { d.Retries = &neg }, "retries"},
		{"no destinations", func(d *DatabaseConfig) { d.Destinations = nil }, "at least one destination"},
		{"unknown destination", func(d *DatabaseConfig) { d.Destinations = []Destination{{Type: "ftp"}} }, "fileServer or local"},
		{"fileServer without token", func(d *DatabaseConfig) {
			d.Destinations = []Destination{{Type: destFileServer, URL: "http://x"}}
		}, "url and token"},
		{"local without path", func(d *DatabaseConfig) { d.Destinations = []Destination{{Type: destLocal}} }, "local needs path"},
		{"negative keepLast", func(d *DatabaseConfig) {
			d.Destinations = []Destination{{Type: destLocal, Path: "x", KeepLast: -1}}
		}, "keepLast"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := validDB()
			tc.mutate(db)
			err := (&Config{Databases: []*DatabaseConfig{db}}).Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	db := validDB()
	db.MysqlConfig.Host = ""
	db.MysqlConfig.Port = 0
	db.Cron = "nope"
	err := (&Config{Databases: []*DatabaseConfig{db}}).Validate()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"MYSQL_HOST", "MYSQL_PORT", "cron"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q is missing %q", err, want)
		}
	}
}

func TestValidateEmptyAndDuplicates(t *testing.T) {
	if err := (&Config{}).Validate(); err == nil {
		t.Error("empty databases should be rejected")
	}
	a, b := validDB(), validDB()
	err := (&Config{Databases: []*DatabaseConfig{a, b}}).Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate name") {
		t.Errorf("error = %v, want duplicate name", err)
	}
	err = (&Config{Databases: []*DatabaseConfig{nil}}).Validate()
	if err == nil || !strings.Contains(err.Error(), "empty entry") {
		t.Errorf("error = %v, want empty entry", err)
	}
}

func TestValidateAcceptsCronSyntax(t *testing.T) {
	for _, spec := range []string{"0 2 * * *", "*/5 * * * *", "@daily", "@every 1h"} {
		db := validDB()
		db.Cron = spec
		if err := (&Config{Databases: []*DatabaseConfig{db}}).Validate(); err != nil {
			t.Errorf("cron %q rejected: %v", spec, err)
		}
	}
}

func TestRetryCount(t *testing.T) {
	zero, five := 0, 5
	d := &DatabaseConfig{}
	if d.RetryCount() != defaultRetries {
		t.Errorf("nil retries should give the default")
	}
	d.Retries = &zero
	if d.RetryCount() != 0 {
		t.Errorf("explicit 0 must be honoured")
	}
	d.Retries = &five
	if d.RetryCount() != 5 {
		t.Errorf("explicit 5 must be honoured")
	}
}

func TestDestinationLabel(t *testing.T) {
	if got := (&Destination{Type: destLocal, Path: `D:\b`}).Label(); got != `local D:\b` {
		t.Errorf("local label = %q", got)
	}
	if got := (&Destination{Type: destFileServer, URL: "http://h/u"}).Label(); got != "fileServer http://h/u" {
		t.Errorf("fileServer label = %q", got)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	good := write("good.json", `{"databases":[{"name":"a",
		"mysqlConfig":{"MYSQL_HOST":"h","MYSQL_PORT":3306,"MYSQL_USERNAME":"u","MYSQL_PASSWORD":"","MYSQL_DATABASE":"d"},
		"destinations":[{"type":"local","path":"x","keepLast":3}]}]}`)
	cfg, err := LoadConfig(good)
	if err != nil {
		t.Fatalf("LoadConfig(good): %v", err)
	}
	if cfg.Databases[0].Destinations[0].KeepLast != 3 {
		t.Errorf("keepLast not parsed")
	}

	if _, err := LoadConfig(filepath.Join(dir, "missing.json")); err == nil || !strings.Contains(err.Error(), "cannot read config") {
		t.Errorf("missing file error = %v", err)
	}
	if _, err := LoadConfig(write("bad.json", `{not json`)); err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("bad JSON error = %v", err)
	}
	if _, err := LoadConfig(write("invalid.json", `{"databases":[{"name":""}]}`)); err == nil || !strings.Contains(err.Error(), "is invalid") {
		t.Errorf("invalid config error = %v", err)
	}
}

// The shipped example must stay loadable (apart from the empty password and
// placeholder URL, which are valid values).
func TestExampleConfigIsValid(t *testing.T) {
	if _, err := LoadConfig("config.json.example"); err != nil {
		t.Fatalf("config.json.example: %v", err)
	}
}
