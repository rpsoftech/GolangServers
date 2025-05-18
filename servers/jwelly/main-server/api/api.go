package jwelly_main_server_apis

import (
	"github.com/gofiber/fiber/v2"
	public "github.com/rpsoftech/golang-servers/servers/jwelly/main-server/api/public"
)

func AddApis(app fiber.Router) {
	public.AddPublicApis(app.Group("/public"))
	// auth.AddAuthPackages(app.Group("/auth"))
	// data.AddDataPackage(app.Group("/data"))
	// order.AddOrderPackage(app.Group("/order"))
}
