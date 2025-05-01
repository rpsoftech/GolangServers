package bullion_main_server_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_auth_apis "github.com/rpsoftech/golang-servers/servers/bullion/main-server/apis/auth"
)

func AddApis(app fiber.Router) {
	bullion_main_server_auth_apis.AddAuthPackages(app.Group("/auth"))
	// data.AddDataPackage(app.Group("/data"))
	// order.AddOrderPackage(app.Group("/order"))
}
