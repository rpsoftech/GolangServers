package soham_common_req_keys

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
)

func ValidateNumberMatchingInToken(c fiber.Ctx, number int) error {
	id, ok := c.Locals(WHATSAPP_CLIENT_NUM_KEY).(int)
	if !ok || number != id {
		return &interfaces.RequestError{
			StatusCode: http.StatusForbidden,
			Code:       ERROR_MISMATCH_NUMBER_FROM_TOKEN,
			Message:    "Your can not access this resource due to different number",
			Name:       "ERROR_MISMATCH_NUMBER_FROM_TOKEN",
			Extra:      fmt.Sprintf("Expected := %d Got := %d", id, number),
		}
	}
	return nil
}
