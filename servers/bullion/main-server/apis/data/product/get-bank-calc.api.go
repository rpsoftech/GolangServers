package bullion_main_server_product_apis

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
)

func apiGetBankCalc(c *fiber.Ctx) error {

	id := c.Query("bullionId")
	if id == "" {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Please Pass Valid Bullion Id",
			Name:       "INVALID_INPUT",
		}
	}
	if err := bullion_main_server_interfaces.ValidateBullionIdMatchingInToken(c, id); err != nil {
		return err
	}
	entity, err := bullion_main_server_services.BankRateCalcService.GetBankRateCalcByBullionId(id)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
