package ecommerce_api_func

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	ecommerce_services "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/services"
)

func GetFilterOptions(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filterService := ecommerce_services.GetFilterService()
	filters, err := filterService.GetFilterOptions(ctx)
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: 500,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Failed to fetch filter options",
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    filters,
	})
}
