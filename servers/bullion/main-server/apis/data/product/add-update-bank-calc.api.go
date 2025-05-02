package bullion_main_server_product_apis

import (
	"github.com/gofiber/fiber/v2"

	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiAddUpdateBankCalcBody struct {
	GOLD_SPOT   *bullion_main_server_interfaces.BankRateCalcBase `bson:"goldSpot" json:"goldSpot" validate:"required"`
	SILVER_SPOT *bullion_main_server_interfaces.BankRateCalcBase `bson:"silverSpot" json:"silverSpot" validate:"required"`
}

func apiAddUpdateBankCalc(c *fiber.Ctx) error {
	body := new(apiAddUpdateBankCalcBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	userId, err := bullion_main_server_interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}
	bullionId, err := bullion_main_server_interfaces.ExtractBullionIdFromCtx(c)
	if err != nil {
		return err
	}
	entity, err := bullion_main_server_services.BankRateCalcService.SaveBankRateCalc(body.GOLD_SPOT, body.SILVER_SPOT, bullionId, userId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
