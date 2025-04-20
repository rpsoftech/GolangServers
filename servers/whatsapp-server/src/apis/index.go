package apis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/middleware"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/whatsapp-http-api/whatsapp"
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
)

func AddApis(app fiber.Router) {
	app.Get("/qr_code", GetQrCode)
	app.Post("/start", StartNumber)
	// app.Get("/qr_scan", QrScan)
	{
		authenticated := app.Group("", middleware.AllowOnlyValidLoggedInWhatsapp)
		authenticated.Post("/send_message", SendMessage)
		authenticated.Post("/send_media", SendMediaFile)
		authenticated.Post("/send_media_64", SendMediaFileWithBase64)
	}
}

func SendMediaFile(c *fiber.Ctx) error {
	body := new(apiSendMessage)
	c.BodyParser(body)
	number, err := interfaces.ExtractNumberFromCtx(c)
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
	connection, ok := whatsapp.ConnectionMap[number]
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
	runHeadLess, err := strconv.ParseBool(interfaces.ExtractKeyFromHeader(c, "Headless"))
	if err != nil {
		runHeadLess = false
	}
	if runHeadLess {
		go connection.SendMediaFileWithPath(body.To, destination, file.Filename, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	} else {
		return c.JSON(connection.SendMediaFileWithPath(body.To, destination, file.Filename, body.Msg))
	}
}
func SendMediaFileWithBase64(c *fiber.Ctx) error {
	body := new(apiSendMediaMsgWithBase64)
	c.BodyParser(body)

	if err := utility.ValidateReqInput(body); err != nil {
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
	number, err := interfaces.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}

	connection, ok := whatsapp.ConnectionMap[number]
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
	runHeadLess, err := strconv.ParseBool(interfaces.ExtractKeyFromHeader(c, "Headless"))
	if err != nil {
		runHeadLess = false
	}
	if runHeadLess {
		go connection.SendMediaFileBase64(body.To, body.Base64, body.FileName, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	} else {
		return c.JSON(connection.SendMediaFileBase64(body.To, body.Base64, body.FileName, body.Msg))
	}
}
func SendMessage(c *fiber.Ctx) error {
	body := new(apiSendMessage)
	c.BodyParser(body)

	if err := utility.ValidateReqInput(body); err != nil {
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
	token, err := interfaces.ExtractNumberFromCtx(c)
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
	connection, ok := whatsapp.ConnectionMap[token]
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
	runHeadLess, err := strconv.ParseBool(interfaces.ExtractKeyFromHeader(c, "Headless"))
	if err != nil {
		runHeadLess = false
	}
	if runHeadLess {
		go connection.SendTextMessage(body.To, body.Msg)
		return c.JSON(fiber.Map{
			"success": true,
		})
	} else {
		return c.JSON(connection.SendTextMessage(body.To, body.Msg))
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
func GetQrCode(c *fiber.Ctx) error {
	number, err := interfaces.ExtractNumberFromCtx(c)
	if err != nil {
		return err
	}

	connection, ok := whatsapp.ConnectionMap[number]
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
