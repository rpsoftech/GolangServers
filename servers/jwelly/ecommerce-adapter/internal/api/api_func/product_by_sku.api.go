package ecommerce_api_func

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	ecommerce_services "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/services"
)

func GetProductBySKU(c fiber.Ctx) error {
	sku := c.Params("sku")
	if sku == "" {
		return &interfaces.RequestError{
			StatusCode: 400,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "SKU parameter is required",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	productService := ecommerce_services.GetProductService()
	product, err := productService.GetProductBySKU(ctx, sku)
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: 404,
			Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
			Message:    "Product not found",
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    product,
	})
}
