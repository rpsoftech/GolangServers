package mysql_to_surreal_interfaces

import (
	"strings"

	"github.com/rpsoftech/golang-servers/validator"
)

type TgMasterStatus string

const (
	TgMasterStatusInStock  TgMasterStatus = "IN STOCK"
	TgMasterStatusApproval TgMasterStatus = "APPROVAL"
	TgMasterStatusModify   TgMasterStatus = "MODIFY"
	TgMasterStatusSold     TgMasterStatus = "SOLD"
)

var tgMasterStatusMap = map[string]TgMasterStatus{
	"IN STOCK": TgMasterStatusInStock,
	"APPROVAL": TgMasterStatusApproval,
	"MODIFY":   TgMasterStatusModify,
	"SOLD":     TgMasterStatusSold,
}

func init() {
	validator.RegisterEnumValidatorFunc("TgMasterStatus", validateEnumTgMasterStatus)
}
func validateEnumTgMasterStatus(value string) bool {
	_, ok := parseTgMasterStatus(value)
	return ok
}
func parseTgMasterStatus(str string) (TgMasterStatus, bool) {
	c, ok := tgMasterStatusMap[strings.ToUpper(str)]
	return c, ok
}
func (e TgMasterStatus) String() string {
	switch e {
	case TgMasterStatusInStock:
		return "IN STOCK"
	case TgMasterStatusApproval:
		return "APPROVAL"
	case TgMasterStatusModify:
		return "MODIFY"
	case TgMasterStatusSold:
		return "SOLD"
	default:
		return ""
	}
}

// var (
// 	appEnvMap = map[string]AppEnv{
// 		"DEVELOPE":   APP_ENV_DEVELOPE,
// 		"LOCAL":      APP_ENV_LOCAL,
// 		"CI":         APP_ENV_CI,
// 		"PRODUCTION": APP_ENV_PRODUCTION,
// 	}
// )
