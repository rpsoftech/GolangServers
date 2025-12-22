package dump_server_apis

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
	dump_server_services "github.com/rpsoftech/golang-servers/servers/http-dump-server/services"
)

func DumpData(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_INVALID_INPUT,
			Message:    "Please Pass Valid UUID Endpoint",
			Name:       "INVALID_INPUT",
		}
	}

	bodyBytes := c.Body()

	// Convert the byte slice to a string if needed
	bodyString := string(bodyBytes)

	// You can then manually unmarshal it using the standard "encoding/json" package
	// var data map[string]interface{}
	// if err := json.Unmarshal(bodyBytes, &data); err != nil {
	// 	return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	// }
	return dump_server_services.EndPointService.DumpData(id, bodyString)

}
