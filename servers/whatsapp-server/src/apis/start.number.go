package whatsapp_server_apis

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	whatsapp_functions "github.com/rpsoftech/golang-servers/functions/whatsapp"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
)

// connectReserved connects a token already reserved in ConnectionMap. It can
// block for minutes while a new device waits for a QR scan. Tests replace it.
var connectReserved = func(jidString string, token string) {
	whatsapp_core.ConnectReserved(jidString, token, whatsapp_core.InitSqlContainer())
}

// jidForToken returns the stored JID for token. Tests replace it.
var jidForToken = func(token string) string {
	return whatsapp_config.WhatsappNumberConfigMap.GetJID(token)
}

// StartNumber starts the WhatsApp connection for the token in the request.
// The connect runs in the background; poll /v1/qr_code for the QR code.
func StartNumber(c fiber.Ctx) error {
	token, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	if !whatsapp_core.ConnectionMap.Reserve(token) {
		return c.JSON(fiber.Map{
			"success": false,
			"reason":  fmt.Sprintf("Number %s is already connected", token),
		})
	}
	go connectReserved(jidForToken(token), token)
	return c.JSON(fiber.Map{
		"success": true,
	})
}
