package bullion_main_server_order_apis

import (
	"github.com/gofiber/fiber/v2"
	adminorder "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/order/admin-order"
	user "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/order/user"
	middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddOrderPackage(router fiber.Router) {
	router.Use(middleware.AllowOnlyValidTokenMiddleWare)
	adminorder.AddAdminOrderRouter(router.Group("/admin", middleware.AllowOnlyBigAdmins.Validate))
	user.AddUserOrderApis(router.Group("/user", middleware.AllowAllAdminsAndTradeUsers.Validate))
}
