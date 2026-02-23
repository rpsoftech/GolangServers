package soham_whatsapp_server_api

import (
	"github.com/gofiber/fiber/v3"
	soham_whatsapp_server_middleware "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/middleware"
)

func AddApis(router fiber.Router) {
	router.Use(soham_whatsapp_server_middleware.TokenDecrypter,
		soham_whatsapp_server_middleware.AllowOnlyValidTokenMiddleWare,
		soham_whatsapp_server_middleware.AllowOnlyValidLoggedInWhatsapp)
	router.Post("/send_message", SendMessageApi)
}
