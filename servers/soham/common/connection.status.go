package soham_common_req_keys

type ConnectionStatus int

const (
	DISCONNECTED  ConnectionStatus = 0
	NOT_LOGGED_IN ConnectionStatus = -1
	LOGGED_IN     ConnectionStatus = 1
)
