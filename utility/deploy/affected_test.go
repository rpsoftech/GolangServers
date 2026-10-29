package main

import (
	"slices"
	"testing"
)

func TestAffectedComponents(t *testing.T) {
	deps := map[string][]string{
		"mysql_backup":          {"servers/jwelly/mysql-backup-cmd", "utility/updater"},
		"mysql_backup_exe":      {"servers/jwelly/mysql-backup-exe", "utility/updater"},
		"whatsapp-server":       {"servers/whatsapp-server", "env"},
		"soham-whatsapp-client": {"servers/soham/whatsapp-client", "env"},
	}
	cases := []struct {
		name    string
		changed []string
		want    []string
	}{
		{"nothing", nil, nil},
		{"unrelated server", []string{"servers/link-shortner/main.go"}, nil},
		{"one component", []string{"servers/jwelly/mysql-backup-exe/dump.go"}, []string{"mysql_backup_exe"}},
		{"shared package", []string{"utility/updater/updater.go"}, []string{"mysql_backup", "mysql_backup_exe"}},
		{"nested file", []string{"servers/whatsapp-server/web/index.html"}, []string{"whatsapp-server"}},
		{"sibling dir with same prefix", []string{"servers/jwelly/mysql-backup-exe-old/x.go"}, nil},
		{"tests and docs ignored", []string{"servers/jwelly/mysql-backup-exe/dump_test.go", "env/README.md"}, nil},
		{"go.mod releases all", []string{"go.mod"}, []string{"mysql_backup", "mysql_backup_exe", "whatsapp-server", "soham-whatsapp-client"}},
		{"deploy tool releases all", []string{"utility/deploy/main.go"}, []string{"mysql_backup", "mysql_backup_exe", "whatsapp-server", "soham-whatsapp-client"}},
		{"windows paths and blanks", []string{"", "env/config.go"}, []string{"whatsapp-server", "soham-whatsapp-client"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := affectedComponents(tc.changed, deps)
			if !slices.Equal(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
