package bullion_main_server_product_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiAddNewProductBody struct {
	*bullion_main_server_interfaces.ProductBaseStruct  `json:"product" validate:"required"`
	*bullion_main_server_interfaces.CalcSnapshotStruct `json:"calcSnapShot" validate:"required"`
}

func apiAddNewProduct(c *fiber.Ctx) error {
	body := new(apiAddNewProductBody)
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
	entity, err := bullion_main_server_services.ProductService.AddNewProduct(body.ProductBaseStruct, body.CalcSnapshotStruct, userId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
