package bullion_main_server_tradeuser_apis

import (
	"github.com/gofiber/fiber/v2"

	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiChangeTradeUserStatusBody struct {
	ID        string `bson:"id" json:"id" validate:"required,uuid"`
	IsActive  bool   `bson:"isActive" json:"isActive" validate:"boolean"`
	BullionId string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
}

func apiChangeTradeUserStatus(c *fiber.Ctx) error {
	body := new(apiChangeTradeUserStatusBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	if err := bullion_main_server_interfaces.ValidateBullionIdMatchingInToken(c, body.BullionId); err != nil {
		return err
	}
	userId, err := bullion_main_server_interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}
	err = bullion_main_server_services.TradeUserService.TradeUserChangeStatus(body.ID, body.BullionId, body.IsActive, userId)
	if err != nil {
		return err
	} else {
		return c.JSON(&fiber.Map{
			"success": true,
			"status":  body.IsActive,
		})
	}

}
