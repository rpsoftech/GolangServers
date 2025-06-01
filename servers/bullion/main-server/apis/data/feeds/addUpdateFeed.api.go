package bullion_main_server_data_feeds_apis

import (
	"github.com/gofiber/fiber/v2"

	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

func apiAddNewFeed(c *fiber.Ctx) error {
	body := new(bullion_main_server_interfaces.FeedsBase)
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
	entity := &bullion_main_server_interfaces.FeedsEntity{
		BaseEntity: &interfaces.BaseEntity{},
		FeedsBase:  body,
	}
	entity.CreateNewId()
	_, err = bullion_main_server_services.FeedsService.AddAndUpdateNewFeeds(entity, userId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
func apiUpdateFeed(c *fiber.Ctx) error {
	body := new(bullion_main_server_interfaces.FeedUpdateRequestBody)
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
	entity, err := bullion_main_server_services.FeedsService.UpdateFeeds(body.FeedsBase, body.FeedId, userId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
