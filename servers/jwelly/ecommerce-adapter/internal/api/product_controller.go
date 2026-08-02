package ecommerce_api

import (
	"github.com/gofiber/fiber/v3"
	ecommerce_api_func "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/internal/api/api_func"
)

func AddApiRoutes(app fiber.Router) {
	RegisterProductRoutes(app.Group("/products"))
	app.Get("/categories", ecommerce_api_func.GetAllCategories)
	app.Get("/product-variants", ecommerce_api_func.GetAllVariants)
	// Master Data APIs
	app.Get("/collections", ecommerce_api_func.GetCollections)
	app.Get("/purities", ecommerce_api_func.GetPurities)
	app.Get("/stone-types", ecommerce_api_func.GetStoneTypes)
	app.Get("/metal-colors", ecommerce_api_func.GetMetalColors)
	app.Get("/occasions", ecommerce_api_func.GetOccasions)
}
func RegisterProductRoutes(app fiber.Router) {
	app.Get("/all", ecommerce_api_func.ProductSearchAll)
	// FIXED: Changed from Get to Post to support JSON request bodies safely
	app.Post("/search", ecommerce_api_func.ProductSearchSearch)
	app.Get("/sku/:sku", ecommerce_api_func.GetProductBySKU)
	app.Get("filters", ecommerce_api_func.GetFilterOptions)
}
