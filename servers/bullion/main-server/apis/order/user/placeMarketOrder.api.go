package bullion_main_server_user_order_apis

import (
	"github.com/gofiber/fiber/v2"
	utility "github.com/rpsoftech/golang-servers/utility/functions"
)

type placeMarketOrderApiBody struct {
	BullionId         string  `json:"bullionId" validate:"required,uuid"`
	UserId            string  `json:"userId" validate:"required,uuid"`
	GroupId           string  `json:"groupId" validate:"required,uuid"`
	Weight            int     `json:"weight" validate:"required,min=0"`
	ProductGroupMapId string  `json:"productGroupMapId" validate:"required,uuid"`
	McxPrice          float64 `json:"mcxPrice" validate:"required,min=0"`
	Price             float64 `json:"price" validate:"required,min=0"`
	// ReqId             float64 `json:"reqId" validate:"required,min=0"`
	// @MessageBody('pgm_id') pgmid: number,
	// @MessageBody('mcx') mcx: number,
	// @MessageBody('price') price: number,
	// @MessageBody('req_id') req_id: number,
}

func PlaceMarkerOrderApi(c *fiber.Ctx) error {
	body := new(placeMarketOrderApiBody)
	c.BodyParser(body)
	if err := utility.ValidateReqInput(body); err != nil {
		return err
	}
	return nil
}
