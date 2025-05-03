package bullion_main_server_data_bank_details_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddBankDetailsAPIs(router fiber.Router) {
	router.Get("/getAll", apiGetBankDetails)
	{
		adminGroup := router.Use(bullion_main_server_middleware.AllowOnlyBigAdmins.Validate)
		adminGroup.Put("/add", apiAddNewBankDetails)
		adminGroup.Patch("/update", apiUpdateBankDetails)
		adminGroup.Delete("/delete", apiDeleteBankDetails)
	}
}
