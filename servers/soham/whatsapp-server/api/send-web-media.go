package soham_whatsapp_server_api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

func (ah *ApiHandler) SendWebMediaApi(c fiber.Ctx) error {
	body := new(soham_common_req_keys.SendWebMediaType)
	if err := c.Bind().Body(body); err != nil {
		return err
	}
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	if err := soham_common_req_keys.ValidateNumberMatchingInToken(c, body.From); err != nil {
		return err
	}
	if !utility_functions.ValidateUrl(body.WebMediaLink) {
		return &interfaces.RequestError{
			StatusCode: fiber.StatusBadRequest,
			Code:       soham_common_req_keys.ERROR_INVALID_WEB_MEDIA_LINK,
			Message:    "The web media link provided is not a valid url",
			Name:       "ERROR_INVALID_WEB_MEDIA_LINK",
		}
	}

	resp, err := ah.whatsappService.SendWebMedia(c.Context(), body)
	if err != nil {
		return err
	}
	return c.JSON(resp)
}
