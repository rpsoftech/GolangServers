package bullion_main_server_apis

import (
	"github.com/gofiber/fiber/v2"
	auth "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/auth"
	data "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/data"
	order "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/order"
)

func AddApis(app fiber.Router) {
	auth.AddAuthPackages(app.Group("/auth"))
	data.AddDataPackage(app.Group("/data"))
	order.AddOrderPackage(app.Group("/order"))
}
