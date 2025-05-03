package bullion_main_server_product_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddProduct(router fiber.Router) {
	router.Get("/getAll", apiGetProducts)
	router.Get("/getProduct", apiGetProducts)
	router.Get("/getBankCalc", apiGetBankCalc)
	adminGroup := router.Use(bullion_main_server_middleware.AllowAllAdmins.Validate)
	adminGroup.Put("/add", apiAddNewProduct)
	adminGroup.Patch("/update", apiUpdateProducts)
	adminGroup.Patch("/updateCalcSnapShot", apiUpdateProductCalcSnapshot)
	adminGroup.Patch("/updateSequence", apiUpdateProductSequence)
	adminGroup.Patch("/updateBankCalc", apiAddUpdateBankCalc)
}

func AddRateApi(router fiber.Router) {
	router.Get("/", apiGetLiveRate)
}
