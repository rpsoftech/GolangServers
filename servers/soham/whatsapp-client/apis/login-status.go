package soham_whatsapp_client_apis

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	whatsapp_functions "github.com/rpsoftech/golang-servers/functions/whatsapp"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	"github.com/rpsoftech/golang-servers/interfaces"
)

func LoginStatus(c *fiber.Ctx) error {
	number, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}

	connection, ok := whatsapp_core.ConnectionMap[number]
	if !ok || connection == nil {
		return &interfaces.RequestError{
			StatusCode: http.StatusNotFound,
			Code:       interfaces.ERROR_CONNECTION_NOT_FOUND,
			Message:    fmt.Sprintf("Number %s Not Found", number),
			Name:       "ERROR_CONNECTION_NOT_FOUND",
		}
	}
	return c.JSON(fiber.Map{
		"status": connection.ConnectionStatus,
	})
}
