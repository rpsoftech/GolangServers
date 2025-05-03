package bullion_main_server_auth_apis

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiTradeUserVerifyOtpBody struct {
	Token string `json:"token"`
	Otp   int    `json:"otp"`
}

func apiTradeUserVerifyOtp(c *fiber.Ctx) error {
	// var body
	body := new(apiTradeUserVerifyOtpBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}

	entity, err := bullion_main_server_services.TradeUserService.VerifyTokenAndVerifyOTP(body.Token, strconv.Itoa(body.Otp))
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
