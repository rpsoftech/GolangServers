package bullion_main_server_auth_apis

import (
	"github.com/gofiber/fiber/v2"
	bullion_main_server_middleware "github.com/rpsoftech/golang-servers/servers/bullion/main-server/middleware"
)

func AddAuthPackages(router fiber.Router) {
	// router.Use(middleware.AllowAllUsers.Validate)
	router.Get("/deviceId", generateDeviceId)
	router.Get("/bullion-details-by-short-name", apiGetBullionDetailsByShortName)
	router.Get("/bullion-details-by-id", apiGetBullionDetailsById)
	{
		generalUserGroup := router.Group("/general-user")
		generalUserGroup.Post("/register", apiRegisterNewGeneralUser)
		generalUserGroup.Get("/get", apiGetGeneralUserDetailsByIdPassword)
		generalUserGroup.Post("/send-for-approval", apiSendApprovalReqGeneralUser)
		generalUserGroup.Post("/get-general-user-token", apiGetGeneralUserToken)
		generalUserGroup.Post("/refresh-token", apiGeneralUSerRefreshToken)
	}
	{
		adminAuthGroup := router.Group("/admin")
		adminAuthGroup.Post("/login", apiAdminLogin)
	}
	{
		tradeUserGroup := router.Group("/trade-user")
		tradeUserGroup.Use(bullion_main_server_middleware.AllowOnlyValidTokenMiddleWare)
		tradeUserGroup.Use(bullion_main_server_middleware.AllowAllUsers.Validate)
		tradeUserGroup.Post("register", apiTradeUserRegister)
		tradeUserGroup.Post("resend-otp", apiTradeUserResendOtp)
		tradeUserGroup.Put("verify-otp", apiTradeUserVerifyOtp)
		tradeUserGroup.Post("login-number", apiTradeUserLoginNumber)
		tradeUserGroup.Post("login-uNumber", apiTradeUserLoginUNumber)
		tradeUserGroup.Post("login-email", apiTradeUserLoginEmail)
	}
}
