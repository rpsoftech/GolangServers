package bullion_main_server_product_apis

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
)

func apiGetProducts(c *fiber.Ctx) (err error) {

	id := c.Query("bullionId")
	if id == "" {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Please Pass Valid Bullion Id",
			Name:       "INVALID_INPUT",
		}
	}
	if err := interfaces.ValidateBullionIdMatchingInToken(c, id); err != nil {
		return err
	}
	productId := c.Query("productId")
	var entity interface{}
	if productId != "" {
		entity, err = bullion_main_server_services.ProductService.GetProductsById(id, productId)
	} else {
		entity, err = bullion_main_server_services.ProductService.GetProductsByBullionId(id)
	}
	if err != nil {
		return err
	} else {
		if productId != "" {
			return c.JSON(&fiber.Map{
				"product": entity,
			})
		} else {
			return c.JSON(&fiber.Map{
				"products": entity,
			})
		}
	}
}
