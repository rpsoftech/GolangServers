package telegram_server

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	telegram_server_config "github.com/rpsoftech/golang-servers/servers/telegram-server/config"
)

func ExtractNumberFromCtx(c *fiber.Ctx) (string, error) {
	id, ok := c.Locals(telegram_server_config.REQ_LOCAL_NUMBER_KEY).(string)
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

func ExtractKeyFromHeader(c *fiber.Ctx, key string) string {
	reqHeaders := c.GetReqHeaders()
	if tokenString, foundToken := reqHeaders[key]; !foundToken || len(tokenString) != 1 || tokenString[0] == "" {
		return ""
	} else {
		return tokenString[0]
	}
}
