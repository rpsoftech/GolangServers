package bullion_main_server_data_bank_details_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

func apiDeleteBankDetails(c *fiber.Ctx) error {
	body := new(bullion_main_server_interfaces.UpdateBankDetailsRequestBody)
	c.BodyParser(body)
	userId, err := bullion_main_server_interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}
	if err := bullion_main_server_interfaces.ValidateBullionIdMatchingInToken(c, body.BullionId); err != nil {
		return err
	}
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	entity, err := bullion_main_server_services.BankDetailsService.DeleteBankDetails(body.BankDetailsBase, body.Id, userId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
