package bullion_main_server_data_bullion_apis

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiUpdateBullionFlagsBody struct {
	BullionId   string                                         `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
	FlagsEntity *bullion_main_server_interfaces.FlagsInterface `bson:"flagsEntity" json:"flagsEntity" validate:"required"`
}

func apiUpdateBullionFlags(c *fiber.Ctx) error {
	body := new(apiUpdateBullionFlagsBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	if err := bullion_main_server_interfaces.ValidateBullionIdMatchingInToken(c, body.BullionId); err != nil {
		return err
	}
	userID, err := bullion_main_server_interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}
	if body.FlagsEntity.BullionId != body.BullionId {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Bullion Id Mismatch",
			Name:       "INVALID_INPUT",
		}
	}
	entity, err := bullion_main_server_services.FlagService.UpdateFlags(body.FlagsEntity, userID)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
