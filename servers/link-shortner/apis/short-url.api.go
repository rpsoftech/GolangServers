package apis

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/interfaces"
)

func GetShortUrlAfterRedirect(c *fiber.Ctx) error {
	id := c.Params("id")
	if len(id) == 0 || strings.Contains(id, ".") {
		return c.Status(400).JSON(interfaces.RequestError{
			Code:    interfaces.ERROR_INVALID_INPUT,
			Message: "Please Pass Valid short URL",
			Name:    "INVALID SHORT URL",
		})
	}
	res, err := firebaseDb.GetShortUrl(id)
	if err != nil || res == nil {
		return c.Status(400).JSON(interfaces.RequestError{
			Code:    interfaces.ERROR_INVALID_INPUT,
			Message: "Please Pass Valid short URL",
			Name:    "INVALID SHORT URL",
		})
	}
	url := res["url"]
	if url == nil {
		return c.Status(400).JSON(interfaces.RequestError{
			Code:    interfaces.ERROR_INVALID_INPUT,
			Message: "Redirect URL Not Found",
			Name:    "INVALID SHORT URL",
		})
	}
	if url, ok := url.(string); !ok {
		return c.Status(400).JSON(interfaces.RequestError{
			Code:    interfaces.ERROR_INVALID_INPUT,
			Message: "Redirect URL Not Found Fount JSON",
			Name:    "INVALID SHORT URL",
		})
	} else {
		return c.Redirect(url)
	}
}
