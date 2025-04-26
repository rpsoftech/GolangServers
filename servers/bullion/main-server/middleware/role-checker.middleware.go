package bullion_main_server_middleware

import (
	"fmt"
	"slices"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type roleCheckerMiddlewareWithRolesArray struct {
	roles []string
}

var (
	allUserRoles = []string{
		bullion_main_server_interfaces.ROLE_RATE_ADMIN.String(),
		bullion_main_server_interfaces.ROLE_SUPER_ADMIN.String(),
		bullion_main_server_interfaces.ROLE_ADMIN.String(),
		bullion_main_server_interfaces.ROLE_GOD.String(),
		bullion_main_server_interfaces.ROLE_GENERAL_USER.String(),
		bullion_main_server_interfaces.ROLE_TRADE_USER.String(),
	}

	AllowAllUsers = &roleCheckerMiddlewareWithRolesArray{
		roles: allUserRoles,
	}
	AllowAllAdminsAndTradeUsers = &roleCheckerMiddlewareWithRolesArray{
		roles: []string{
			bullion_main_server_interfaces.ROLE_ADMIN.String(),
			bullion_main_server_interfaces.ROLE_RATE_ADMIN.String(),
			bullion_main_server_interfaces.ROLE_SUPER_ADMIN.String(),
			bullion_main_server_interfaces.ROLE_GOD.String(),
			bullion_main_server_interfaces.ROLE_TRADE_USER.String(),
		},
	}
	AllowAllAdmins = &roleCheckerMiddlewareWithRolesArray{
		roles: []string{
			bullion_main_server_interfaces.ROLE_ADMIN.String(),
			bullion_main_server_interfaces.ROLE_RATE_ADMIN.String(),
			bullion_main_server_interfaces.ROLE_SUPER_ADMIN.String(),
			bullion_main_server_interfaces.ROLE_GOD.String(),
		},
	}
	AllowOnlyBigAdmins = &roleCheckerMiddlewareWithRolesArray{
		roles: []string{
			bullion_main_server_interfaces.ROLE_ADMIN.String(),
			bullion_main_server_interfaces.ROLE_SUPER_ADMIN.String(),
			bullion_main_server_interfaces.ROLE_GOD.String(),
		},
	}
)

func (cc *roleCheckerMiddlewareWithRolesArray) Validate(c *fiber.Ctx) error {
	roleFromLocal := c.Locals(interfaces.REQ_LOCAL_KEY_ROLE)
	if roleFromLocal == nil {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_TOKEN_ROLE_NOT_FOUND,
			Message:    "Invalid Token Role Or Not Found",
			Name:       "INVALID_TOKEN_ROLE",
		}
	}
	role, ok := roleFromLocal.(string)
	if !ok {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_ROLE_NOT_EXISTS,
			Message:    "Token Role Should be string",
			Name:       "INVALID_TOKEN_ROLE_FORMAT",
		}
	}
	if role == string(bullion_main_server_interfaces.ROLE_GOD) {
		return c.Next()
	}
	if !slices.Contains(cc.roles, role) {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_ROLE_NOT_AUTHORIZED,
			Message:    fmt.Sprintf("%s is not allowed for this route", role),
			Name:       "INVALID_TOKEN_ROLE_FORMAT",
		}
	}
	return c.Next()
}
