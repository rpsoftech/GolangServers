package bullion_main_server_auth_apis

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
)

func apiGeneralUSerRefreshToken(c *fiber.Ctx) error {
	var body map[string]string
	json.Unmarshal(c.Body(), &body)
	token, ok := body["token"]
	if !ok {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Please Pass Valid Token",
			Name:       "INVALID_INPUT",
		}
	}
	entity, err := bullion_main_server_services.GeneralUserService.RefreshToken(token)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
