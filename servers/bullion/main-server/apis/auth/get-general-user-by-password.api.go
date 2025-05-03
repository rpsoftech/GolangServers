package bullion_main_server_auth_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type getGeneralUserBody struct {
	Id       string `json:"id" validate:"required,uuid"`
	Password string `json:"password" validate:"required"`
}

func apiGetGeneralUserDetailsByIdPassword(c *fiber.Ctx) error {
	body := new(getGeneralUserBody)
	c.QueryParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	entity, err := bullion_main_server_services.GeneralUserService.GetGeneralUserDetailsByIdPassword(body.Id, body.Password)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
