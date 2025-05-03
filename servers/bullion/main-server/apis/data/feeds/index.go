package bullion_main_server_data_feeds_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddFeedsAndNotificationSection(router fiber.Router) {
	router.Get("/getPaginated", apiGetFeedsApiPaginated)
	{
		adminGroup := router.Use(bullion_main_server_middleware.AllowOnlyBigAdmins.Validate)
		adminGroup.Post("/sendNotification", apiSendFeedAsNotification)
		adminGroup.Post("/add", apiAddNewFeed)
		adminGroup.Patch("/update", apiUpdateFeed)
		adminGroup.Delete("/delete", apiDeleteFeeds)
	}
}
