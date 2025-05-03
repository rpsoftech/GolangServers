package whatsapp_server_apis

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	whatsapp_config "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/config"
	whatsapp_functions "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/functions"
	"github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/whatsapp"
)

func StartNumber(c *fiber.Ctx) error {
	token, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	_, ok := whatsapp.ConnectionMap[token]
	if ok {
		return c.JSON(fiber.Map{
			"success": false,
			"reason":  fmt.Sprintf("Number %s is already connected", token),
		})
	}
	jidString := whatsapp_config.WhatsappNumberConfigMap.JID[token]
	whatsapp.ConnectToNumber(jidString, token)
	return c.JSON(fiber.Map{
		"success": true,
	})
}
