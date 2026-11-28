package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
	"github.com/rpsoftech/golang-servers/servers/pm2go/daemon"
)

// call sends one request to the daemon, spawning it when absent.
func call(method string, params any, out any) error {
	conn, err := connect(true)
	if err != nil {
		return err
	}
	defer conn.Close()
	return roundTrip(conn, method, params, out)
}

func roundTrip(conn net.Conn, method string, params any, out any) error {
	req := common.Request{Method: method}
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		req.Params = b
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return err
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return err
	}
	var resp common.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return errors.New(resp.Error)
	}
	if out != nil && len(resp.Data) > 0 {
		return json.Unmarshal(resp.Data, out)
	}
	return nil
}

// dial opens a connection to the daemon. Tests replace it.
var dial = defaultDial

// spawn starts a detached daemon. Tests replace it.
var spawn = spawnDaemon

// spawnWait is how long connect waits for a freshly spawned daemon.
var spawnWait = 5 * time.Second

func defaultDial() (net.Conn, error) {
	return net.DialTimeout("unix", common.SocketPath(), time.Second)
}

func connect(autoSpawn bool) (net.Conn, error) {
	conn, err := dial()
	if err == nil {
		return conn, nil
	}
	if !autoSpawn {
		return nil, errors.New("daemon is not running")
	}
	if err := spawn(); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(spawnWait)
	for time.Now().Before(deadline) {
		if conn, err := dial(); err == nil {
			return conn, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("daemon did not start, see %s", common.DaemonLogFile())
}

func spawnDaemon() error {
	if err := common.EnsureDirs(); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(common.DaemonLogFile(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logf.Close()
	cmd := exec.Command(exe, "daemon")
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.SysProcAttr = daemon.DetachAttr()
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
