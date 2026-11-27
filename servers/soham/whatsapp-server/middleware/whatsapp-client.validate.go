package soham_whatsapp_server_middleware

import (
	"crypto/subtle"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
	if err := checkClientToken(soham_whatsapp_server_env.Env.UUIDObj, numberToken, uuidToken); err != nil {
		return err
	}
	c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY, numberToken)
	c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY, uuidToken)
	return c.Next()
}

// checkClientToken checks that numberToken is a positive number in canonical
// form ("7", not "007" or "+7") and that uuidToken is the UUIDv5 of it under
// base. The UUID compare is constant time.
func checkClientToken(base uuid.UUID, numberToken, uuidToken string) error {
	if n, err := strconv.Atoi(numberToken); err != nil || n <= 0 || strconv.Itoa(n) != numberToken {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       soham_common_req_keys.ERROR_NUMBER_TOKEN_NOT_PASSED,
			Message:    "Please Pass Valid Number Token",
			Name:       "ERROR_NUMBER_TOKEN_NOT_PASSED",
		}
	}
	generateToken := utility_functions.UUIDv5(base, numberToken)
	if subtle.ConstantTimeCompare([]byte(generateToken), []byte(uuidToken)) != 1 {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       soham_common_req_keys.ERROR_GENERATE_TOKEN_MISMATCH,
			Message:    "Please PASS VALID UUID AND NUMBER TOKEN",
			Name:       "ERROR_GENERATE_TOKEN_MISMATCH",
		}
	}
	return nil
}
