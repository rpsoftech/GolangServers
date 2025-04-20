package whatsapp_functions

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_config "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/config"
)

func ExtractKeyFromHeader(c *fiber.Ctx, key string) string {
	reqHeaders := c.GetReqHeaders()
	if tokenString, foundToken := reqHeaders[key]; !foundToken || len(tokenString) != 1 || tokenString[0] == "" {
		return ""
	} else {
		return tokenString[0]
	}
}
func ExtractNumberFromCtx(c *fiber.Ctx) (string, error) {
	id, ok := c.Locals(whatsapp_config.REQ_LOCAL_NUMBER_KEY).(string)
	if !ok {
		return "", &interfaces.RequestError{
			StatusCode: http.StatusForbidden,
			Code:       interfaces.INVALID_NUMBER_FROM_TOKEN,
			Message:    "Invalid Number From Token",
			Name:       "INVALID_NUMBER_FROM_TOKEN",
		}
	}
	return id, nil
}
