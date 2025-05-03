package bullion_main_server_auth_apis

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_services "github.com/rpsoftech/golang-servers/servers/bullion/main-server/services"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type apiTradeUserLoginUNumberBody struct {
	UNumber  string `bson:"uNumber" json:"uNumber" validate:"required"`
	Password string `bson:"rawPassword" json:"password" mapstructure:"password" validate:"required,min=4"`
}
type apiTradeUserLoginNumberBody struct {
	Number   string `bson:"number" json:"number" validate:"required,min=12,max=12"`
	Password string `bson:"rawPassword" json:"password" mapstructure:"password" validate:"required,min=4"`
}
type apiTradeUserLoginEmailBody struct {
	Email    string `bson:"email" json:"email" validate:"required"`
	Password string `bson:"rawPassword" json:"password" mapstructure:"password" validate:"required,min=4"`
}

func apiTradeUserLoginNumber(c *fiber.Ctx) error {
	body := new(apiTradeUserLoginNumberBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	bullionId, err := interfaces.ExtractBullionIdFromCtx(c)
	if err != nil {
		return err
	}
	entity, err := bullion_main_server_services.TradeUserService.LoginWithNumberAndPassword(body.Number, body.Password, bullionId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
func apiTradeUserLoginUNumber(c *fiber.Ctx) error {
	body := new(apiTradeUserLoginUNumberBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	bullionId, err := interfaces.ExtractBullionIdFromCtx(c)
	if err != nil {
		return err
	}
	entity, err := bullion_main_server_services.TradeUserService.LoginWithUNumberAndPassword(body.UNumber, body.Password, bullionId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
func apiTradeUserLoginEmail(c *fiber.Ctx) error {
	body := new(apiTradeUserLoginEmailBody)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	bullionId, err := interfaces.ExtractBullionIdFromCtx(c)
	if err != nil {
		return err
	}
	entity, err := bullion_main_server_services.TradeUserService.LoginWithEmailAndPassword(body.Email, body.Password, bullionId)
	if err != nil {
		return err
	} else {
		return c.JSON(entity)
	}
}
