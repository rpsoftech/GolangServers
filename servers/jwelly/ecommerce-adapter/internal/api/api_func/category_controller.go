package ecommerce_api_func

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	ecommerce_services "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/services"
)

func GetAllCategories(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	categoryService := ecommerce_services.GetCategoryService()
	categories, err := categoryService.GetAllCategories(ctx)
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: 500,
			Code:       interfaces.ERROR_INTERNAL_SERVER,
			Message:    "Failed to fetch categories",
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    categories,
	})
}
