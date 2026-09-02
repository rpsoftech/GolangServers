package soham_whatsapp_server_api

import (
	"github.com/gofiber/fiber/v3"
	soham_whatsapp_server_middleware "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/middleware"
	soham_whatsapp_server_services "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/services"
)

type ApiHandler struct {
	whatsappService *soham_whatsapp_server_services.WhatsappService
}

func AddApis(router fiber.Router) {
	tokenValidator := soham_whatsapp_server_middleware.GetTokenValidator()
	router.Use(tokenValidator.TokenDecrypter,
		tokenValidator.AllowOnlyValidTokenMiddleWare,
		tokenValidator.AllowOnlyValidLoggedInWhatsapp)
	apiHandler := &ApiHandler{
		whatsappService: soham_whatsapp_server_services.GetWhatsappService(),
	}
	router.Post("/send_message", apiHandler.SendMessageApi)
	router.Post("/send_base64_media", apiHandler.SendBase64ImageApi)
	router.Post("/send_web_media", apiHandler.SendWebMediaApi)
	router.Post("/send_local_media", apiHandler.SendLocalMediaApi)
}
