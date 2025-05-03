package bullion_main_server_auth_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

func apiGetGeneralUserToken(c *fiber.Ctx) error {
	body := new(addApprovalReqGeneralUserBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	entity, err := bullion_main_server_services.GeneralUserService.ValidateApprovalAndGenerateToken(body.Id, body.Password, body.BullionId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
