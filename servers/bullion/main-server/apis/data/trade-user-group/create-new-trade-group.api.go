package bullion_main_server_tradeusergroup_apis

import (
	"github.com/gofiber/fiber/v2"

	interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiCreateNewTradeGroupBody struct {
	Name      string `bson:"name" json:"name" validate:"required"`
	BullionId string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
}

func apiCreateNewTradeGroup(c *fiber.Ctx) error {
	body := new(apiCreateNewTradeGroupBody)
	c.BodyParser(body)
	if err := utility.ValidateReqInput(body); err != nil {
		return err
	}
	if err := interfaces.ValidateBullionIdMatchingInToken(c, body.BullionId); err != nil {
		return err
	}
	userID, err := interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}
	entity, err := services.TradeUserGroupService.CreateNewTradeUserGroup(body.BullionId, body.Name, userID)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}

}
