package apis

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	link_shorner_utility "github.com/rpsoftech/golang-servers/servers/link-shortner/link-shorner-utility"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	// "github.com/keyurboss/Generic-Servers/go/link-shortner/firebase"
	// "github.com/keyurboss/Generic-Servers/go/link-shortner/interfaces"
)

type bodyBulkShortning struct {
	Urls []string `json:"urls" validate:"required,min=1"`
}

type bodySingleShortning struct {
	Urls string `json:"url" validate:"required"`
}

func BulkShortning(c *fiber.Ctx) error {
	body := new(bodyBulkShortning)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	resp := make(map[string]string)
	for _, url := range body.Urls {
		if !utility_functions.ValidateUrl(url) {
			continue
		}
		id := link_shorner_utility.UniqueIdConst.GetUniqueId()
		resp[id] = link_shorner_utility.ServerUrl + id
		firebaseDb.SetPublicData(id, map[string]interface{}{"url": url, "created_at": time.Now().Unix()})
	}
	return c.JSON(resp)
}

func SignleShortning(c *fiber.Ctx) error {
	body := new(bodySingleShortning)
	c.BodyParser(body)
	if err := utility_functions.ValidateReqInput(body); err != nil {
		return err
	}
	resp := make(map[string]string)

	// for _, url := range body.Urls {
	if !utility_functions.ValidateUrl(body.Urls) {
		return c.Status(400).JSON(interfaces.RequestError{
			Code:    interfaces.ERROR_INVALID_INPUT,
			Message: "Please Pass Valid URL",
			Name:    "INVALID URL",
		})
	}
	id := link_shorner_utility.UniqueIdConst.GetUniqueId()
	resp[id] = link_shorner_utility.ServerUrl + id
	firebaseDb.SetPublicData(id, map[string]interface{}{"url": body.Urls, "created_at": time.Now().Unix()})
	// }
	return c.JSON(resp)
}
