package bullion_main_server_product_apis

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

func apiUpdateProducts(c *fiber.Ctx) error {
	body := new([]bullion_main_server_interfaces.UpdateProductApiBody)
	c.BodyParser(body)
	if len(*body) == 0 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Please pass at least 1 product to update",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
	userId, err := bullion_main_server_interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}

	bullionId, err := bullion_main_server_interfaces.ExtractBullionIdFromCtx(c)
	if err != nil {
		return err
	}
	for _, ele := range *body {
		if err := utility_functions.ValidateReqInput(&ele); err != nil {
			return err
		}
		if ele.BullionId != bullionId {
			return &interfaces.RequestError{
				StatusCode: 403,
				Code:       interfaces.ERROR_PERMISSION_NOT_ALLOWED,
				Message:    "You can not change other resources",
				Name:       "ERROR_BULLION_ID_MISMATCH",
				Extra: fiber.Map{
					"bullionId": bullionId,
					"product":   ele,
				},
			}
		}
	}
	if entity, err := bullion_main_server_services.ProductService.UpdateProduct(body, userId); err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
func apiUpdateProductCalcSnapshot(c *fiber.Ctx) error {
	body := new([]bullion_main_server_interfaces.UpdateProductCalcSnapshotApiBody)
	c.BodyParser(body)
	if len(*body) == 0 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Please pass at least 1 product to update",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
	userId, err := bullion_main_server_interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}

	bullionId, err := bullion_main_server_interfaces.ExtractBullionIdFromCtx(c)
	if err != nil {
		return err
	}
	for _, ele := range *body {
		if err := utility_functions.ValidateReqInput(&ele); err != nil {
			return err
		}
		if ele.BullionId != bullionId {
			return &interfaces.RequestError{
				StatusCode: 403,
				Code:       interfaces.ERROR_PERMISSION_NOT_ALLOWED,
				Message:    "You can not change other resources",
				Name:       "ERROR_BULLION_ID_MISMATCH",
				Extra: fiber.Map{
					"bullionId": bullionId,
					"product":   ele,
				},
			}
		}
	}
	if entity, err := bullion_main_server_services.ProductService.UpdateProductCalcSnapshot(body, userId); err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
func apiUpdateProductSequence(c *fiber.Ctx) error {
	body := new([]bullion_main_server_interfaces.UpdateProductCalcSequenceApiBody)
	c.BodyParser(body)
	if len(*body) < 2 {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Please pass at least 2 product to update",
			Name:       "ERROR_INVALID_INPUT",
		}
	}
	userId, err := bullion_main_server_interfaces.ExtractTokenUserIdFromCtx(c)
	if err != nil {
		return err
	}

	bullionId, err := bullion_main_server_interfaces.ExtractBullionIdFromCtx(c)
	if err != nil {
		return err
	}
	for _, ele := range *body {
		if err := utility_functions.ValidateReqInput(&ele); err != nil {
			return err
		}
		if ele.BullionId != bullionId {
			return &interfaces.RequestError{
				StatusCode: 403,
				Code:       interfaces.ERROR_PERMISSION_NOT_ALLOWED,
				Message:    "You can not change other resources",
				Name:       "ERROR_BULLION_ID_MISMATCH",
				Extra: fiber.Map{
					"bullionId": bullionId,
					"product":   ele,
				},
			}
		}
	}
	if entity, err := bullion_main_server_services.ProductService.UpdateProductSequence(body, userId, bullionId); err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
