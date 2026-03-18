package soham_whatsapp_client_apis

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	whatsapp_functions "github.com/rpsoftech/golang-servers/functions/whatsapp"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/skip2/go-qrcode"
)

func GetQrCode(c *fiber.Ctx) error {
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
	err = connection.ReturnStatusError()

	if err != nil {
		png, _ := qrcode.Encode(connection.QrCodeString, qrcode.High, 512)
		return c.JSON(fiber.Map{
			"qrCode":     base64.StdEncoding.EncodeToString(png),
			"qrCodeData": connection.QrCodeString,
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
	})
}
