package soham_whatsapp_server_api

import (
	"github.com/gofiber/fiber/v3"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_services "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

func SendLocalMediaApi(c fiber.Ctx) error {
	body := new(soham_common_req_keys.SendFilePathMediaType)
	// c.BodyParser(body)
	if err := c.Bind().Body(body); err != nil {
		return err
	}
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	if err := soham_common_req_keys.ValidateNumberMatchingInToken(c, body.From); err != nil {
		return err
	}
	resp, err := soham_whatsapp_server_services.WhatsappService.SendLocalMedia(body)
	if err != nil {
		return err
	}
	return c.JSON(resp)
}
