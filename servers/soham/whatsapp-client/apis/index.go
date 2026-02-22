package soham_whatsapp_client_apis

import (
	"github.com/gofiber/fiber/v2"
)

func AddApis(app fiber.Router) {
	app.Get("/qr_code", GetQrCode)
}
