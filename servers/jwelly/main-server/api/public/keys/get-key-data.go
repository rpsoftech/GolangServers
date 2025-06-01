package jwelly_main_server_apis_public_keys

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	jwelly_main_server_repos "github.com/rpsoftech/golang-servers/servers/jwelly/main-server/repos"
)

func GetKeyData(c *fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Please Pass Valid Key",
			Name:       "INVALID_INPUT",
		}
	}
	entity, err := jwelly_main_server_repos.KeyValueDataRepo.FindOneByKey(key)
	if err != nil {
		return err
	}
	return c.JSON(entity)
}
