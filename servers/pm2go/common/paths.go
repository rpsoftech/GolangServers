package common

import (
	"os"
	"path/filepath"
	"regexp"
)

// Home returns the state directory: $PMGO_HOME or ~/.pm2go.
func Home() string {
	if h := os.Getenv("PMGO_HOME"); h != "" {
		return h
	}
	u, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "pm2go")
	}
	return filepath.Join(u, ".pm2go")
}

func LogsDir() string       { return filepath.Join(Home(), "logs") }
func SocketPath() string    { return filepath.Join(Home(), "rpc.sock") }
func PidFile() string       { return filepath.Join(Home(), "pm2go.pid") }
func DumpFile() string      { return filepath.Join(Home(), "dump.json") }
func DaemonLogFile() string { return filepath.Join(Home(), "pm2go.log") }

func EnsureDirs() error {
	if err := os.MkdirAll(LogsDir(), 0o755); err != nil {
		return err
	}
	return nil
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// SafeName makes a process name usable as a file name.
func SafeName(name string) string { return unsafeName.ReplaceAllString(name, "_") }
