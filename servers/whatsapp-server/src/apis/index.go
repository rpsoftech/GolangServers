package whatsapp_server_apis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
		apiSendMediaMsgWithWebLink
		URL []string `json:"urls" validate:"required,url"`
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
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Invalid Request Body",
			Name:       "ERROR_INVALID_INPUT",
			Extra:      err,
		}
	}

	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}

	if len(body.To) == 0 || len(body.To[0]) < 7 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Number Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}

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

	if err := connection.ReturnStatusError(); err != nil {
		return err
	}

	go connection.SendMediaFileFromURLs(context.Background(), body.To, body.URL, body.FileName, body.Msg)
	return c.JSON(fiber.Map{
		"success": true,
		"msg":     "Media are queued for sending",
	})

}
func SendMediaFileWithWebLink(c fiber.Ctx) error {
	body := new(apiSendMediaMsgWithWebLink)
	if err := c.Bind().Body(body); err != nil {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Invalid Request Body",
			Name:       "ERROR_INVALID_INPUT",
			Extra:      err,
		}
	}

	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}

	if len(body.To) == 0 || len(body.To[0]) < 7 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Number Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}

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

	if err := connection.ReturnStatusError(); err != nil {
		return err
	}

	runHeadLess, err := strconv.ParseBool(whatsapp_functions.ExtractKeyFromHeader(c, "Headless"))
	if err != nil {
		runHeadLess = false
	}

	if runHeadLess {
		go connection.SendMediaFileFromURL(context.Background(), body.To, body.URL, body.FileName, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		return c.JSON(connection.SendMediaFileFromURL(ctx, body.To, body.URL, body.FileName, body.Msg))
	}
}

func SendMediaFile(c fiber.Ctx) error {
	body := new(apiSendMessage)
	c.Bind().Body(body)
	number, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "File Not Found",
			Name:       "ERROR_INVALID_INPUT",
			Extra:      err,
		}
	}
	json.Unmarshal([]byte(c.FormValue("to", "[]")), &body.To)
	json.Unmarshal([]byte(c.FormValue("msg", "")), &body.Msg)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}

	if len(body.To) == 0 || len(body.To[0]) < 7 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Number Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
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
		return err
	}

	destination := fmt.Sprintf("./tmp/%s", file.Filename)
	if err := c.SaveFile(file, destination); err != nil {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Error While Saving File",
			Name:       "ERROR_INTERNAL_SERVER",
			Extra:      err,
		}
	}
	runHeadLess, err := strconv.ParseBool(whatsapp_functions.ExtractKeyFromHeader(c, "Headless"))
	if err != nil {
		runHeadLess = false
	}
	if runHeadLess {

		go connection.SendMediaFileWithPath(context.Background(), body.To, destination, file.Filename, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		return c.JSON(connection.SendMediaFileWithPath(ctx, body.To, destination, file.Filename, body.Msg))
	}
}
func SendMediaFileWithBase64(c fiber.Ctx) error {
	body := new(apiSendMediaMsgWithBase64)
	c.Bind().Body(body)

	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}

	if len(body.To) == 0 || len(body.To[0]) < 7 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Number Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
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
		return err
	}
	runHeadLess, err := strconv.ParseBool(whatsapp_functions.ExtractKeyFromHeader(c, "Headless"))
	if err != nil {
		runHeadLess = false
	}
	if runHeadLess {
		go connection.SendMediaFileBase64(context.Background(), body.To, body.Base64, body.FileName, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		return c.JSON(connection.SendMediaFileBase64(ctx, body.To, body.Base64, body.FileName, body.Msg))
	}
}
func SendMessage(c fiber.Ctx) error {
	body := new(apiSendMessage)
	c.Bind().Body(body)

	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	if len(body.To) == 0 || len(body.To[0]) < 7 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Number Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
	token, err := whatsapp_functions.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}
	if len(body.Msg) == 0 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Message Not Found",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
	connection, ok := whatsapp_core.ConnectionMap[token]
	if !ok || connection == nil {
		return &interfaces.RequestError{
			StatusCode: http.StatusNotFound,
			Code:       interfaces.ERROR_CONNECTION_NOT_FOUND,
			Message:    fmt.Sprintf("Number %s Not Found", token),
			Name:       "ERROR_CONNECTION_NOT_FOUND",
		}
	}
	err = connection.ReturnStatusError()
	if err != nil {
		return err
	}
	runHeadLess, err := strconv.ParseBool(whatsapp_functions.ExtractKeyFromHeader(c, "Headless"))
	if err != nil {
		runHeadLess = false
	}
	if runHeadLess {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		go connection.SendTextMessage(ctx, body.To, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	} else {
		ctx, cancel := context.WithTimeout(c.Context(), time.Second*30)
		defer cancel()
		return c.JSON(connection.SendTextMessage(ctx, body.To, body.Msg))
	}
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
