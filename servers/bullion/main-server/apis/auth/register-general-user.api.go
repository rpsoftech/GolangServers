package bullion_main_server_auth_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type registerGeneralUserBody struct {
	BullionId string      `json:"bullionId" validate:"required,uuid"`
	User      interface{} `json:"user" validate:"required"`
}

func apiRegisterNewGeneralUser(c *fiber.Ctx) error {
	body := new(registerGeneralUserBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	entity, err := bullion_main_server_services.GeneralUserService.RegisterNew(body.BullionId, body.User)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
