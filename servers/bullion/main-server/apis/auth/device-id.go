package bullion_main_server_auth_apis

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func generateDeviceId(c *fiber.Ctx) error {
	id := uuid.New()
	return c.SendString(strings.ReplaceAll(id.String(), "-", ""))
}
