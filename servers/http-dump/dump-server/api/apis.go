package dump_server_apis

import (
	"github.com/gofiber/fiber/v2"
)

func AddApis(app fiber.Router) {
	app.All("/dump/:id<guid>", DumpData)
	// auth.AddAuthPackages(app.Group("/auth"))
	// data.AddDataPackage(app.Group("/data"))
	// order.AddOrderPackage(app.Group("/order"))
}
