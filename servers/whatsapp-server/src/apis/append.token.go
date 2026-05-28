package whatsapp_server_apis

import (
	"github.com/gofiber/fiber/v3"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
)

func AppendTokenInConfigJSON(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.JSON(fiber.Map{
			"success": false,
		})
	}
	// check token exist in config
	if _, ok := whatsapp_config.WhatsappNumberConfigMap.Tokens[token]; ok {
		return c.JSON(fiber.Map{
			"success": false,
		})
	}
	whatsapp_config.WhatsappNumberConfigMap.Tokens[token] = ""
	whatsapp_config.WhatsappNumberConfigMap.Save()
	return c.JSON(fiber.Map{
		"success": true,
	})
}
