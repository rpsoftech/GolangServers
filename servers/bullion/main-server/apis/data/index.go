package bullion_main_server_data_apis

import (
	"github.com/gofiber/fiber/v2"
	bankdetails "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/data/bank-details"
	bullion "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/data/bullion"
	feeds "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/data/feeds"
	product "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/data/product"
	tradeuser "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/data/trade-user"
	tradeusergroup "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/data/trade-user-group"
	middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddDataPackage(router fiber.Router) {
	router.Use(middleware.AllowOnlyValidTokenMiddleWare)
	router.Use(middleware.AllowAllUsers.Validate)
	product.AddRateApi(router.Group("/rates"))
	{
		productGroup := router.Group("/product")
		product.AddProduct(productGroup)
	}
	{
		feedGroup := router.Group("/feeds")
		feeds.AddFeedsAndNotificationSection(feedGroup)
	}
	{
		bankGroup := router.Group("/bank-details")
		bankdetails.AddBankDetailsAPIs(bankGroup)
	}
	{
		tradeUserRoute := router.Group("/trade-user")
		tradeuser.AddTradeUserAPIs(tradeUserRoute)
	}
	{
		tradeUserGroupGroup := router.Group("/tradeUserGroup")
		tradeusergroup.AddTradeUserGroupAPIs(tradeUserGroupGroup)
	}
	{
		bullion.AddBullionApis(router.Group("/bullion-details"))
	}
}
