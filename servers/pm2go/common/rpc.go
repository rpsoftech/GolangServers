package common

import "encoding/json"

// RPC methods. The wire format is one JSON object per line over a unix socket.
const (
	MethodPing      = "ping"
	MethodStart     = "start"
	MethodStop      = "stop"
	MethodRestart   = "restart"
	MethodDelete    = "delete"
	MethodList      = "list"
	MethodDescribe  = "describe"
	MethodSave      = "save"
	MethodResurrect = "resurrect"
	MethodFlush     = "flush"
	MethodKill      = "kill"
	MethodReload    = "reload"
	MethodScale     = "scale"
	MethodLink      = "link"
	MethodSystem    = "system"
)

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// StartParams starts (or updates and restarts) one app. CLIEnv is the caller's
// environment; it is used for the child but never written to the dump file.
type StartParams struct {
	Config AppConfig `json:"config"`
	CLIEnv []string  `json:"cli_env,omitempty"`
}

// TargetParams selects processes by numeric id, name, namespace or "all".
type TargetParams struct {
	Target string `json:"target"`
}

type ScaleParams struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
