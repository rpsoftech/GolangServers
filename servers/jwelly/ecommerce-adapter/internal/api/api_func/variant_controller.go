package ecommerce_api_func

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	ecommerce_services "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/services"
)

func GetAllVariants(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		return &interfaces.RequestError{
			StatusCode: 400,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Invalid Offset",
		}
	}

	variantService := ecommerce_services.GetVariantService()
	variants, err := variantService.GetPaginatedVariants(ctx, limit, offset)
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: 500,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Failed to fetch variants",
		}
	}

	page := (offset / limit) + 1

	return c.JSON(fiber.Map{
		"success": true,
		"page":    page,
		"limit":   limit,
		"total":   len(variants),
		"data":    variants,
	})
}
