package ecommerce_api_func

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	ecommerce_services "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/services"
)

func ProductSearchAll(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: 400,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Invalid Limit",
			Name:       "ERROR_INVALID_LIMIT",
		}
	}
	// Prevent division by zero below
	if limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		return &interfaces.RequestError{
			StatusCode: 400,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Invalid Offset",
			Name:       "ERROR_INVALID_OFFSET",
		}
	}

	productService := ecommerce_services.GetProductService()
	products, err := productService.GetAllProductsForWirewings(ctx, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// FIXED: Dynamically calculate the actual page number based on offset and limit
	page := (offset / limit) + 1

	return c.JSON(fiber.Map{
		"success": true,
		"page":    page,
		"limit":   limit,
		"total":   len(products),
		"data":    products,
	})
}
