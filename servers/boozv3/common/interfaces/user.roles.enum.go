package booz_main_interfaces

import (
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/validator"
)

type UserRoles string

const (
	ROLE_GOD          UserRoles = "GOD"
	ROLE_SUPER_ADMIN  UserRoles = "SUPER_ADMIN"
	ROLE_ADMIN        UserRoles = "ADMIN"
	ROLE_SPOT_MANAGER UserRoles = "ROLE_SPOT_MANAGER"
	ROLE_SPOT_COUNTER UserRoles = "ROLE_SPOT_COUNTER"
	ROLE_SPOT_FLEET   UserRoles = "ROLE_SPOT_FLEET"
	ROLE_GENERAL_USER UserRoles = "GENERAL_USER"
)

var (
	userRolesMap = interfaces.EnumValidatorBase{
		Data: map[string]interface{}{
			"GOD":               ROLE_GOD,
			"SUPER_ADMIN":       ROLE_SUPER_ADMIN,
			"ADMIN":             ROLE_ADMIN,
			"ROLE_SPOT_MANAGER": ROLE_SPOT_MANAGER,
			"ROLE_SPOT_COUNTER": ROLE_SPOT_COUNTER,
			"ROLE_SPOT_FLEET":   ROLE_SPOT_FLEET,
			"GENERAL_USER":      ROLE_GENERAL_USER,
		},
	}
)

func init() {
	validator.RegisterEnumValidatorFunc("UserRoles", userRolesMap.Validate)
}

func ValidateEnumUserRole(value string) bool {
	return userRolesMap.Validate(value)
}

func (s UserRoles) String() string {
	switch s {
	case ROLE_GOD:
		return "GOD"
	case ROLE_SUPER_ADMIN:
		return "SUPER_ADMIN"
	case ROLE_ADMIN:
		return "ADMIN"
	case ROLE_SPOT_MANAGER:
		return "ROLE_SPOT_MANAGER"
	case ROLE_SPOT_COUNTER:
		return "ROLE_SPOT_COUNTER"
	case ROLE_SPOT_FLEET:
		return "ROLE_SPOT_FLEET"
	case ROLE_GENERAL_USER:
		return "GENERAL_USER"

	}
	return "unknown"
}

func (s UserRoles) IsValid() bool {
	switch s {
	case
		ROLE_GOD,
		ROLE_SUPER_ADMIN,
		ROLE_ADMIN,
		ROLE_SPOT_MANAGER,
		ROLE_SPOT_COUNTER,
		ROLE_SPOT_FLEET,
		ROLE_GENERAL_USER:
		return true
	}
	return false
}
