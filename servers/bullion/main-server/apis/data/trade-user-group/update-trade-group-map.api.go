package bullion_main_server_tradeusergroup_apis

import (
	"github.com/gofiber/fiber/v2"

	interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiUpdateTradeUserGroupMapBody struct {
	Base      *[]interfaces.TradeUserGroupMapEntity `bson:"base" json:"base" validate:"required"`
	GroupId   string                                `bson:"groupId" json:"groupId" validate:"required,uuid"`
	BullionId string                                `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
}

func apiUpdateTradeUserGroupMap(c *fiber.Ctx) error {
	body := new(apiUpdateTradeUserGroupMapBody)
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
	entity, err := services.TradeUserGroupService.UpdateTradeGroupMap(body.Base, body.GroupId, body.BullionId, userID)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}

}
