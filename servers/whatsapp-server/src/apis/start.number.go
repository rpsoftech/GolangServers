package whatsapp_server_apis

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	whatsapp_functions "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/functions"
)

func StartNumber(c fiber.Ctx) error {
	token, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	_, ok := whatsapp_core.ConnectionMap[token]
	if ok {
		return c.JSON(fiber.Map{
			"success": false,
			"reason":  fmt.Sprintf("Number %s is already connected", token),
		})
	}
	jidString := whatsapp_config.WhatsappNumberConfigMap.JID[token]
	whatsapp_core.ConnectToNumber(jidString, token, whatsapp_core.InitSqlContainer())
	return c.JSON(fiber.Map{
		"success": true,
	})
}
