package bullion_main_server_tradeuser_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
)

func apiGetInActiveTradeUsers(c *fiber.Ctx) error {
	bullionId, err := bullion_main_server_interfaces.ExtractBullionIdFromCtx(c)
	if err != nil {
		return err
	}
	entity, err := bullion_main_server_services.TradeUserService.FindAndReturnAllInActiveTradeUsers(bullionId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}

}
