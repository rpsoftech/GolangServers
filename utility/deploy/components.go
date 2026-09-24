package main

import (
	"fmt"

	"github.com/rpsoftech/golang-servers/utility/updater"
)

// Target is one GOOS/GOARCH build.
type Target struct{ OS, Arch string }

func (t Target) String() string { return t.OS + "_" + t.Arch }

// Component is one publishable program. Adding a program to the release
// pipeline means adding an entry here; its code calls utility/updater with
// the same Name.
type Component struct {
	// Name is the component part of the KV key <ENV>_<Name>_<os>_<arch>.
	Name string
	// Package is the main package, relative to the repo root.
	Package string
	// Binary is the executable name inside the published .gz (".exe" is
	// added for Windows).
	Binary  string
	Targets []Target
	// LegacyKey, if set, returns an extra KV key that is also written for
	// PRODUCTION releases, so installs built before the unified pipeline can
	// still find the new release. Remove it once those installs are gone.
	LegacyKey func(t Target) string
}

var (
	serverTargets = []Target{
		{"linux", "amd64"},
		{"windows", "amd64"},
	}
	desktopTargets = []Target{
		{"windows", "amd64"},
		{"windows", "arm64"},
		{"windows", "386"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
	}
)

var components = []Component{
	{
		Name:    updater.MysqlBackupCmdProjectName,
		Package: "./servers/jwelly/mysql-backup-cmd",
		Binary:  "mysql-backup-cmd",
		Targets: serverTargets,
	},
	{
		Name:    updater.WhatsappProjectName,
		Package: "./servers/whatsapp-server",
		Binary:  "whatsapp-server",
		Targets: serverTargets,
	},
	{
		Name:    updater.SohamWhatsappClientProjectName,
		Package: "./servers/soham/whatsapp-client",
		Binary:  "whatsapp-client",
		Targets: desktopTargets,
		// Soham clients and launchers released by the old release/soham tool
		// read soham_go_wbot_<os>_<arch>.
		LegacyKey: func(t Target) string { return fmt.Sprintf("soham_go_wbot_%s_%s", t.OS, t.Arch) },
	},
}

func findComponent(name string) (Component, bool) {
	for _, c := range components {
		if c.Name == name {
			return c, true
		}
	}
	return Component{}, false
}
