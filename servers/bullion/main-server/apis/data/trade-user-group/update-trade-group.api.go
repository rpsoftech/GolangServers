package bullion_main_server_tradeusergroup_apis

import (
	"github.com/gofiber/fiber/v2"

	interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiUpdateTradeUserGroupBody struct {
	Base    *interfaces.TradeUserGroupBase `bson:"base" json:"base" validate:"required"`
	GroupId string                         `bson:"groupId" json:"groupId" validate:"required,uuid"`
}

func apiUpdateTradeUserGroup(c *fiber.Ctx) error {
	body := new(apiUpdateTradeUserGroupBody)
	c.BodyParser(body)
	if err := utility.ValidateReqInput(body); err != nil {
		return err
	}
	if err := interfaces.ValidateBullionIdMatchingInToken(c, body.Base.BullionId); err != nil {
		return err
	}
	userID, err := interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}
	entity, err := services.TradeUserGroupService.UpdateTradeGroup(body.Base, body.GroupId, userID)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}

}
