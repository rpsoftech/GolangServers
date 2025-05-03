package bullion_main_server_data_bullion_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddBullionApis(router fiber.Router) {
	// app.Get("/bullions", getBullions)
	{
		adminGroup := router.Group("/admin", bullion_main_server_middleware.AllowOnlyBigAdmins.Validate)
		adminGroup.Patch("/updateBullionFlags", apiUpdateBullionFlags)
	}
	{
		tradeUserGroup := router.Group("/user")
		tradeUserGroup.Get("/getBullionFlags", apiGetBullionFlags)
	}
}
