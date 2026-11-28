package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

// Version is set at build time with -ldflags "-X .../cli.Version=...".
var Version = "dev"

const unitDir = "/etc/systemd/system"

func unitName(kind string) string { return "pm2go-" + kind + ".service" }

// cmdStartup installs (install=true) or removes a systemd unit. kind is
// "daemon" (default) or "hub".
func cmdStartup(args []string, install bool) error {
	kind := "daemon"
	var extra []string
	if len(args) > 0 {
		kind, extra = args[0], args[1:]
	}
	if kind != "daemon" && kind != "hub" {
		return fmt.Errorf("unknown target %q, want daemon or hub", kind)
	}
	name := unitName(kind)
	path := filepath.Join(unitDir, name)
	if !install {
		_ = exec.Command("systemctl", "disable", "--now", name).Run()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		_ = exec.Command("systemctl", "daemon-reload").Run()
		fmt.Println("removed", path)
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return err
	}
	unit := renderUnit(kind, exe, u.Username, common.Home(), extra)
	if os.Geteuid() != 0 {
		fmt.Printf("Not root. Save this as %s, then run:\n  systemctl daemon-reload && systemctl enable --now %s\n\n%s", path, name, unit)
		return errors.New("rerun with sudo to install automatically")
	}
	if err := os.WriteFile(path, []byte(unit), 0o644); err != nil {
		return err
	}
	for _, c := range [][]string{{"daemon-reload"}, {"enable", "--now", name}} {
		if out, err := exec.Command("systemctl", c...).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl %s: %v: %s", strings.Join(c, " "), err, out)
		}
	}
	fmt.Println("installed and started", name)
	return nil
}

func renderUnit(kind, exe, username, home string, extra []string) string {
	args := "daemon --resurrect"
	if kind == "hub" {
		args = strings.TrimSpace("hub " + strings.Join(extra, " "))
	}
	// The daemon unit uses KillMode=process: systemd signals only the daemon, so
	// managed applications survive a daemon restart and the next daemon adopts
	// them. There is no ExecStop that kills them (it would also run on restart);
	// `pm2go kill` stops them. The hub has no children and keeps control-group.
	killMode := "process"
	if kind == "hub" {
		killMode = "control-group"
	}
	return fmt.Sprintf(`[Unit]
Description=pm2go %s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%s
Environment=PMGO_HOME=%s
ExecStart=%s %s
Restart=always
RestartSec=3
KillMode=%s
TimeoutStopSec=30
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`, kind, username, home, exe, args, killMode)
}
