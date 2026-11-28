package daemon

import "errors"

// Windows is a development target only: adoption needs a process start time
// and command line, which are not read here, so nothing is ever adopted.

var errNoProcInfo = errors.New("process info is not available on windows")

func procStartTime(pid int) (string, error) { return "", errNoProcInfo }
func procCmdline(pid int) (string, error)   { return "", errNoProcInfo }
func processAlive(pid int) bool             { return false }
