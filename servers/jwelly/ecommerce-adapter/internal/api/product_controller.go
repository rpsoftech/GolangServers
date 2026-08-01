package ecommerce_api

import (
	"github.com/gofiber/fiber/v3"
	ecommerce_api_func "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/internal/api/api_func"
)

func RegisterProductRoutes(app *fiber.App) {
	app.Get("/api/products/all", ecommerce_api_func.ProductSearchAll)
	app.Get("/api/products/search", ecommerce_api_func.ProductSearchSearch)
}
