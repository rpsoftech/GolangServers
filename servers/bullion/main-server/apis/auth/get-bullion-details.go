package bullion_main_server_auth_apis

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
)

func apiGetBullionDetailsByShortName(c *fiber.Ctx) error {
	var body map[string]string = c.Queries()
	shortName, ok := body["name"]
	if !ok || len(shortName) < 2 {
		err := &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Invalid Short Name",
			Name:       "ERROR_INVALID_INPUT",
		}
		return err
	}
	if entity, err := bullion_main_server_services.BullionDetailsService.GetBullionDetailsByShortName(shortName); err != nil {
		return err
	} else {
		return c.JSON(entity.BullionPublicInfo)
	}
}

func apiGetBullionDetailsById(c *fiber.Ctx) error {
	var body map[string]string = c.Queries()
	bullionId, ok := body["id"]
	if !ok || len(bullionId) < 2 {
		err := &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Invalid Short Name",
			Name:       "ERROR_INVALID_INPUT",
		}
		return err
	}
	if entity, err := bullion_main_server_services.BullionDetailsService.GetBullionDetailsByBullionId(bullionId); err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
