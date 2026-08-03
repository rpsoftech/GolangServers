package ecommerce_api_func

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	ecommerce_services "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/services"
)

// 1. Collections (Mirroring Categories as discussed)
func GetCollections(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	categories, err := ecommerce_services.GetCategoryService().GetAllCategories(ctx)
	if err != nil {
		return &interfaces.RequestError{StatusCode: 500, Code: interfaces.ERROR_INTERNAL_SERVER, Message: "Failed to fetch collections"}
	}
	return c.JSON(fiber.Map{"success": true, "data": categories})
}

// 2. Purities (Extracted from our existing Filter Service)
func GetPurities(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filters, err := ecommerce_services.GetFilterService().GetFilterOptions(ctx)
	if err != nil {
		return &interfaces.RequestError{StatusCode: 500, Code: interfaces.ERROR_INTERNAL_SERVER, Message: "Failed to fetch purities"}
	}
	return c.JSON(fiber.Map{"success": true, "data": filters.Purities})
}

// 3. Hardcoded Mock APIs for Wirewings UI Development
func GetStoneTypes(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "data": []string{"Diamond", "Ruby", "Emerald", "Sapphire", "Cubic Zirconia"}})
}

func GetMetalColors(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "data": []string{"Yellow Gold", "Rose Gold", "White Gold"}})
}

func GetOccasions(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "data": []string{"Bridal", "Anniversary", "Daily Wear", "Festival"}})
}
