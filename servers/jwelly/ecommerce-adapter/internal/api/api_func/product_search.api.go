package ecommerce_api_func

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	ecommerce_dto "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/interfaces/dto"
	ecommerce_services "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/services"
)

func ProductSearchSearch(c fiber.Ctx) error {
	var searchReq ecommerce_dto.ProductSearchRequest
	if err := c.Bind().Query(&searchReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request payload",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productService := ecommerce_services.GetProductService()
	products, err := productService.SearchAndFilterProducts(ctx, &searchReq)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	page := searchReq.Page
	if page <= 0 {
		page = 1
	}
	limit := searchReq.Limit
	if limit <= 0 {
		limit = 20
	}

	return c.JSON(fiber.Map{
		"success": true,
		"page":    page,
		"limit":   limit,
		"total":   len(products),
		"data":    products,
	})
}
