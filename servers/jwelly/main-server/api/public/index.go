package jwelly_main_server_apis_public

import (
	"github.com/gofiber/fiber/v2"
	jwelly_main_server_apis_public_keys "github.com/rpsoftech/golang-servers/servers/jwelly/main-server/api/public/keys"
)

func AddPublicApis(router fiber.Router) {
	router.Get("/get-key", jwelly_main_server_apis_public_keys.GetKeyData)
}
