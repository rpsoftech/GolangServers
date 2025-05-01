package bullion_main_server_auth_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiAdminLoginBody struct {
	UserName  string `json:"uname" validate:"required"`
	Password  string `json:"password" validate:"required"`
	BullionId string `json:"bullionId" validate:"required,uuid"`
}

func apiAdminLogin(c *fiber.Ctx) error {
	body := new(apiAdminLoginBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	entity, err := bullion_main_server_services.AdminUserService.ValidateUserAndGenerateToken(body.UserName, body.Password, body.BullionId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
