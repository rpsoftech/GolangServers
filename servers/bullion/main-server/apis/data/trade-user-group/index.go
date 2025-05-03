package bullion_main_server_tradeusergroup_apis

import (
	"github.com/gofiber/fiber/v2"
	middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddTradeUserGroupAPIs(router fiber.Router) {
	// router.
	// adminGroup := router.Group("/admin", middleware.AllowOnlyBigAdmins.Validate)
	{
		adminGroup := router.Group("/admin", middleware.AllowOnlyBigAdmins.Validate)
		adminGroup.Put("/createNewTradeGroup", apiCreateNewTradeGroup)
		adminGroup.Patch("/updateTradeGroup", apiUpdateTradeUserGroup)
		adminGroup.Patch("/updateTradeGroupProductMap", apiUpdateTradeUserGroupMap)
	}
	adminAndTradeGroup := router.Use(middleware.AllowAllAdminsAndTradeUsers.Validate)
	adminAndTradeGroup.Get("/getTradeGroupDetailsByID", apiGetTradeGroupDetailsById)
	adminAndTradeGroup.Get("/getTradeGroupDetailsByBullionId", apiGetTradeGroupDetailsByBullionId)
	adminAndTradeGroup.Get("/getTradeGroupMapDetailsByGroupId", apiGetTradeGroupMapDetailsByGroupId)
}
