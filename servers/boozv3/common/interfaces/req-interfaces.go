package booz_main_interfaces

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
)

const (
	REQ_LOCAL_KEY_ROLE           = "UserRole"
	REQ_LOCAL_ERROR_KEY          = "Error"
	REQ_LOCAL_UserID             = "UserId"
	REQ_LOCAL_KEY_TOKEN_RAW_DATA = "TokenRawData"
)

func ExtractTokenUserIdFromCtx(c *fiber.Ctx) (string, error) {
	id, ok := c.Locals(REQ_LOCAL_UserID).(string)
	if !ok {
		return "", &interfaces.RequestError{
			StatusCode: http.StatusForbidden,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Your can not access this resource due to different UserId",
			Name:       "ERROR_USER_ID_NOT_FOUND",
		}
	}
	return id, nil
}
