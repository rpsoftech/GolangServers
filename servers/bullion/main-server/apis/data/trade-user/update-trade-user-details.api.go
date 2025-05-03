package bullion_main_server_tradeuser_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
)

func apiUpdateTradeUserDetails(c *fiber.Ctx) error {
	body := new(bullion_main_server_interfaces.TradeUserEntity)
	c.BodyParser(body)
	if err := bullion_main_server_interfaces.ValidateBullionIdMatchingInToken(c, body.BullionId); err != nil {
		return err
	}
	userId, err := bullion_main_server_interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}
	err = bullion_main_server_services.TradeUserService.UpdateTradeUser(body, userId)
	if err != nil {
		return err
	} else {
		return c.JSON(&fiber.Map{
			"success": true,
		})
	}

}
