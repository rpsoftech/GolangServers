package bullion_main_server_interfaces

import (
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/validator"
)

type GeneralUserAuthStatus string

const (
	GENERAL_USER_AUTH_STATUS_AUTHORIZED GeneralUserAuthStatus = "Authorized"
	GENERAL_USER_AUTH_STATUS_REQUESTED  GeneralUserAuthStatus = "Requested"
	GENERAL_USER_AUTH_STATUS_REJECTED   GeneralUserAuthStatus = "Rejected"
)

var (
	generalUserAuthStatusMap = interfaces.EnumValidatorBase{
		Data: map[string]interface{}{
			"Authorized": GENERAL_USER_AUTH_STATUS_AUTHORIZED,
			"Requested":  GENERAL_USER_AUTH_STATUS_REQUESTED,
			"Rejected":   GENERAL_USER_AUTH_STATUS_REJECTED,
		},
	}
)

func init() {
	validator.RegisterEnumValidatorFunc("GeneralUserAuthStatus", generalUserAuthStatusMap.Validate)
}
func (s GeneralUserAuthStatus) String() string {
	switch s {
	case GENERAL_USER_AUTH_STATUS_AUTHORIZED:
		return "Authorized"
	case GENERAL_USER_AUTH_STATUS_REQUESTED:
		return "Requested"
	case GENERAL_USER_AUTH_STATUS_REJECTED:
		return "Rejected"
	}
	return "unknown"
}

func (s GeneralUserAuthStatus) IsValid() bool {
	switch s {
	case
		GENERAL_USER_AUTH_STATUS_AUTHORIZED,
		GENERAL_USER_AUTH_STATUS_REQUESTED,
		GENERAL_USER_AUTH_STATUS_REJECTED:
		return true
	}

	return false
}
