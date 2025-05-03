package bullion_main_server_product_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
)

func apiGetLiveRate(c *fiber.Ctx) error {
	return c.JSON(bullion_main_server_services.LiveRateService.LastRateMap)
}
