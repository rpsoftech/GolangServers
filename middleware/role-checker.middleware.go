package middlewares

import (
	"fmt"
	"slices"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
)

var REQ_LOCAL_KEY_ROLE = "UserRole"
var ROLE_GOD = "GOD"

type RoleCheckerMiddlewareWithRolesArray struct {
	roles []string
}

func SetVariablesForRoleCheckerMiddleware(rEQ_LOCAL_KEY_ROLE string, rOLE_GOD string) {
	REQ_LOCAL_KEY_ROLE = rEQ_LOCAL_KEY_ROLE
	ROLE_GOD = rOLE_GOD

}

func (cc *RoleCheckerMiddlewareWithRolesArray) Validate(c *fiber.Ctx) error {
	roleFromLocal := c.Locals(REQ_LOCAL_KEY_ROLE)
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
	if role == string(ROLE_GOD) {
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
