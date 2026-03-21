package soham_common_req_keys

import (
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/validator"
)

type ConnectionStatus string

const (
	DISCONNECTED  ConnectionStatus = "0"
	NOT_LOGGED_IN ConnectionStatus = "-1"
	LOGGED_IN     ConnectionStatus = "1"
)

var (
	connectionStatusMap = interfaces.EnumValidatorBase{
		Data: map[string]any{
			"0":  DISCONNECTED,
			"-1": NOT_LOGGED_IN,
			"1":  LOGGED_IN,
		},
	}
)

func init() {
	validator.RegisterEnumValidatorFunc("ConnectionStatusEnum", connectionStatusMap.Validate)
}
func StringToEnumConnectionStatus(value string) (ConnectionStatus, bool) {
	switch value {
	case "0":
		return DISCONNECTED, true
	case "-1":
		return NOT_LOGGED_IN, true
	case "1":
		return LOGGED_IN, true
	default:
		return DISCONNECTED, false
	}
}

func (s ConnectionStatus) String() string {
	switch s {
	case DISCONNECTED:
		return "0"
	case NOT_LOGGED_IN:
		return "-1"
	case LOGGED_IN:
		return "1"
	}
	return "0"
}
func (s ConnectionStatus) IsValid() bool {
	switch s {
	case DISCONNECTED, NOT_LOGGED_IN, LOGGED_IN:
		return true
	}
	return false
}
