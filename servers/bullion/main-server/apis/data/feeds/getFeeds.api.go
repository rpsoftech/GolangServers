package bullion_main_server_data_feeds_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiGetFeedsApiPaginatedBody struct {
	BullionId string `json:"bullionId" validate:"required,uuid"`
	Page      int64  `json:"page" validate:"required,min=1"`
	Limit     int64  `json:"limit" validate:"required,min=1"`
}

func apiGetFeedsApiPaginated(c *fiber.Ctx) error {
	body := new(apiGetFeedsApiPaginatedBody)
	c.QueryParser(body)
	if err := bullion_main_server_interfaces.ValidateBullionIdMatchingInToken(c, body.BullionId); err != nil {
		return err
	}
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	entity, err := bullion_main_server_services.FeedsService.FetchPaginatedFeedsByBullionId(body.BullionId, body.Page-1, body.Limit)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
