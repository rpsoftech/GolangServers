package soham_whatsapp_server_middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

func ValidateWhatsAppClientToken(c fiber.Ctx) error {
	uuidToken := c.Get(soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY)
	if uuidToken == "" {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       soham_common_req_keys.ERROR_INVALID_UUID_TOKEN,
			Message:    "Please Pass Valid UUID Token",
			Name:       "ERROR_TOKEN_NOT_BEFORE",
		}
	}
	numberToken := c.Get(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY)
	if numberToken == "" {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       soham_common_req_keys.ERROR_NUMBER_TOKEN_NOT_PASSED,
			Message:    "Please Pass Valid Number Token",
			Name:       "ERROR_NUMBER_TOKEN_NOT_PASSED",
		}
	}
	generateToken := utility_functions.UUIDv5(soham_whatsapp_server_env.Env.UUIDObj, numberToken)
	if generateToken != uuidToken {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       soham_common_req_keys.ERROR_GENERATE_TOKEN_MISMATCH,
			Message:    "Please PASS VALID UUID AND NUMBER TOKEN",
			Name:       "ERROR_GENERATE_TOKEN_MISMATCH",
		}
	}
	c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY, numberToken)
	c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY, uuidToken)
	return c.Next()
}
