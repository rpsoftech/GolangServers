package whatsapp_server_apis

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v3"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_functions "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/functions"
	whatsapp_server_middleware "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/middleware"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/skip2/go-qrcode"
)

type (
	apiSendMessage struct {
		To  []string `json:"to" validate:"required,dive,min=1"`
		Msg string   `json:"msg"`
	}
	apiSendMediaMsgWithBase64 struct {
		apiSendMessage
		FileName string `json:"fileName" validate:"required,min=3"`
		Base64   string `json:"base64" validate:"required,min=3"`
	}
	apiSendMediaMsgWithWebLinks struct {
		apiSendMessage
		URL []string `json:"urls" validate:"required,min=1,dive,http_url"`
	}
	apiSendMediaMsgWithWebLink struct {
		apiSendMessage
		URL      string `json:"url" validate:"required,url"`
		FileName string `json:"fileName"` // Optional, fallback extracted from URL if empty
	}
)

func AddApis(app fiber.Router) {
	app.Get("/qr_code", GetQrCode)
	app.Post("/start", StartNumber)
	// app.Get("/qr_scan", QrScan)
	{
		authenticated := app.Group("", whatsapp_server_middleware.AllowOnlyValidLoggedInWhatsapp)
		authenticated.Post("/send_message", SendMessage)
		authenticated.Post("/send_media", SendMediaFile)
		authenticated.Post("/send_media_64", SendMediaFileWithBase64)
		authenticated.Post("/send_media_url", SendMediaFileWithWebLink)           // NEW Endpoint
		authenticated.Post("/send_media_url_multiple", SendMediaFileWithWebLinks) // NEW Endpoint
	}
}

func SendMediaFileWithWebLinks(c fiber.Ctx) error {
	body := new(apiSendMediaMsgWithWebLinks)
	if err := c.Bind().Body(body); err != nil {
		return invalidBody(err)
	}
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	to, err := normalizeRecipients(body.To)
	if err != nil {
		return err
	}
	number, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	connection, err := lookupConnection(number)
	if err != nil {
		return err
	}

	// Each URL is sent to every recipient, so the budget covers both.
	timeout := sendTimeout(len(to) * len(body.URL))
	go func(urls []string, msg string) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		connection.SendMediaFileFromURLs(ctx, to, urls, msg)
	}(body.URL, body.Msg)
	return c.JSON(fiber.Map{
		"success": true,
		"msg":     "Media are queued for sending",
	})
}

func SendMediaFileWithWebLink(c fiber.Ctx) error {
	body := new(apiSendMediaMsgWithWebLink)
	if err := c.Bind().Body(body); err != nil {
		return invalidBody(err)
	}
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	to, err := normalizeRecipients(body.To)
	if err != nil {
		return err
	}
	number, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	connection, err := lookupConnection(number)
	if err != nil {
		return err
	}

	timeout := sendTimeout(len(to))
	if isHeadless(c) {
		go func(url, fileName, msg string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			connection.SendMediaFileFromURL(ctx, to, url, fileName, msg)
		}(body.URL, body.FileName, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := connection.SendMediaFileFromURL(ctx, to, body.URL, body.FileName, body.Msg)
	if result == nil {
		return sendFailed(http.StatusBadGateway, "Error While Downloading File From URL")
	}
	return c.JSON(result)
}

func SendMediaFile(c fiber.Ctx) error {
	// This endpoint is multipart only. Fields are read from the form
	// directly: `to` may be a JSON array or a comma-separated list and `msg`
	// is plain text, which the generic body binder does not handle.
	number, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return invalidInput("File Not Found", err.Error())
	}
	body := new(apiSendMessage)
	if body.To, err = parseRecipientsField(c.FormValue("to")); err != nil {
		return err
	}
	body.Msg = parseMsgField(c.FormValue("msg"))
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	to, err := normalizeRecipients(body.To)
	if err != nil {
		return err
	}
	connection, err := lookupConnection(number)
	if err != nil {
		return err
	}

	destination, err := saveUpload(file)
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: http.StatusInternalServerError,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Error While Saving File",
			Name:       "ERROR_INTERNAL_SERVER",
			Extra:      err.Error(),
		}
	}
	timeout := sendTimeout(len(to))
	if isHeadless(c) {
		go func(fileName, msg string) {
			defer removeUpload(destination)
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			connection.SendMediaFileWithPath(ctx, to, destination, fileName, msg)
		}(file.Filename, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	}
	defer removeUpload(destination)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := connection.SendMediaFileWithPath(ctx, to, destination, file.Filename, body.Msg)
	if result == nil {
		return sendFailed(http.StatusInternalServerError, "Error While Reading Uploaded File")
	}
	return c.JSON(result)
}

func removeUpload(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("could not remove upload %s: %v", path, err)
	}
}

func SendMediaFileWithBase64(c fiber.Ctx) error {
	body := new(apiSendMediaMsgWithBase64)
	if err := c.Bind().Body(body); err != nil {
		return invalidBody(err)
	}
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	to, err := normalizeRecipients(body.To)
	if err != nil {
		return err
	}
	number, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	connection, err := lookupConnection(number)
	if err != nil {
		return err
	}

	timeout := sendTimeout(len(to))
	if isHeadless(c) {
		go func(data, fileName, msg string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			connection.SendMediaFileBase64(ctx, to, data, fileName, msg)
		}(body.Base64, body.FileName, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := connection.SendMediaFileBase64(ctx, to, body.Base64, body.FileName, body.Msg)
	if result == nil {
		return sendFailed(http.StatusBadRequest, "Invalid Base64 Data")
	}
	return c.JSON(result)
}

func SendMessage(c fiber.Ctx) error {
	body := new(apiSendMessage)
	if err := c.Bind().Body(body); err != nil {
		return invalidBody(err)
	}
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	to, err := normalizeRecipients(body.To)
	if err != nil {
		return err
	}
	token, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	if len(body.Msg) == 0 {
		return invalidInput("Message Not Found", nil)
	}
	connection, err := lookupConnection(token)
	if err != nil {
		return err
	}

	timeout := sendTimeout(len(to))
	if isHeadless(c) {
		// The context must belong to the goroutine. A deferred cancel in the
		// handler fires on return and would cancel the send before it starts.
		go func(msg string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			connection.SendTextMessage(ctx, to, msg)
		}(body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	}
	ctx, cancel := context.WithTimeout(c.Context(), timeout)
	defer cancel()
	return c.JSON(connection.SendTextMessage(ctx, to, body.Msg))
}

// GetQrCode returns the QR Code of the connection based on the number in the request context
// @Summary Returns the QR Code of the connection
// @Description Returns the QR Code of the connection based on the number in the request context
// @Tags Connection
// @Accept  json
// @Produce  json
// @Param number path string true "Number"
// @Success 200 {object} fiber.Map{qrCode string}
// @Success 200 {object} fiber.Map{success bool}
// @Failure 404 {object} interfaces.RequestError
// @Failure 500 {object} interfaces.RequestError
// @Router /connections/{number}/qrcode [get]
func GetQrCode(c fiber.Ctx) error {
	number, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}

	connection, ok := whatsapp_core.ConnectionMap.Get(number)
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
		png, _ := qrcode.Encode(connection.QRCode(), qrcode.High, 512)
		return c.JSON(fiber.Map{
			"qrCode":     base64.StdEncoding.EncodeToString(png),
			"qrCodeData": connection.QRCode(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
	})
}
