package bullion_main_server_data_bank_details_apis

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
)

func apiGetBankDetails(c *fiber.Ctx) error {
	bullionId := c.Query("bullionId")
	if bullionId == "" {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "bullionId is required",
			Name:       "INVALID_INPUT",
		}
	}
	if err := interfaces.ValidateBullionIdMatchingInToken(c, bullionId); err != nil {
		return err
	}
	entity, err := bullion_main_server_services.BankDetailsService.GetBankDetailsByBullionId(bullionId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
