package daemon

import (
	"fmt"
	"os"
	"strings"
)

// procStat returns the fields of /proc/<pid>/stat after the command name:
// index 0 is the state (field 3), index 19 the start time (field 22).
func procStatFields(pid int) ([]string, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return nil, err
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')') // the name may contain spaces and parentheses
	if i < 0 || i+2 > len(s) {
		return nil, fmt.Errorf("bad /proc/%d/stat", pid)
	}
	f := strings.Fields(s[i+2:])
	if len(f) < 20 {
		return nil, fmt.Errorf("short /proc/%d/stat", pid)
	}
	return f, nil
}

// procStartTime returns the process start time in clock ticks since boot, as
// text. A reused pid has a different value.
func procStartTime(pid int) (string, error) {
	f, err := procStatFields(pid)
	if err != nil {
		return "", err
	}
	return f[19], nil
}

func procCmdline(pid int) (string, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.ReplaceAll(string(b), "\x00", " ")), nil
}

// processAlive reports whether the pid exists and is not a zombie.
func processAlive(pid int) bool {
	f, err := procStatFields(pid)
	return err == nil && f[0] != "Z" && f[0] != "X"
}
