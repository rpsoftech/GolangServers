package bullion_main_server_tradeuser_apis

import (
	"github.com/gofiber/fiber/v2"
	middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddTradeUserAPIs(router fiber.Router) {
	adminGroup := router.Use(middleware.AllowOnlyBigAdmins.Validate)
	adminGroup.Get("/getInActiveTradeUsers", apiGetInActiveTradeUsers)
	adminGroup.Get("/getTradeUser", apiGetTradeUserDetails)
	adminGroup.Patch("/updateTradeUserDetails", apiUpdateTradeUserDetails)
	adminGroup.Patch("/updateTradeUserStatus", apiChangeTradeUserStatus)
}
