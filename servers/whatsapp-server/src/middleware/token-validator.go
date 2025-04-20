package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_config "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/config"
)

const (
	REQ_LOCAL_KEY_ROLE           = "UserRole"
	REQ_LOCAL_ERROR_KEY          = "Error"
	REQ_LOCAL_NUMBER_KEY         = "Number"
	REQ_LOCAL_KEY_TOKEN_RAW_DATA = "TokenRawData"
)

// fiber middleware for jwt
func TokenDecrypter(c *fiber.Ctx) error {
	reqHeaders := c.GetReqHeaders()
	println(c.IP())
	tokenString, foundToken := reqHeaders[env.RequestTokenHeaderKey]
	if !foundToken || len(tokenString) != 1 || tokenString[0] == "" {
		if c.IP() == "127.0.0.1" || whatsapp_config.Env.ALLOW_LOCAL_NO_AUTH {
			tokenString = []string{"default"}
		} else {

			c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, &interfaces.RequestError{
				StatusCode: 403,
				Code:       interfaces.ERROR_TOKEN_NOT_BEFORE,
				Message:    "Please Pass Valid Token",
				Name:       "ERROR_TOKEN_NOT_PASSED",
			})
			return c.Next()
		}
	}
	if _, ok := whatsapp_config.WhatsappNumberConfigMap.Tokens[tokenString[0]]; !ok {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Invalid Token",
			Name:       "ERROR_INVALID_TOKEN",
		})
	} else {
		c.Locals(REQ_LOCAL_NUMBER_KEY, tokenString[0])
	}
	return c.Next()
}

func AllowOnlyValidTokenMiddleWare(c *fiber.Ctx) error {
	jwtRawFromLocal := c.Locals(REQ_LOCAL_NUMBER_KEY)
	localError := c.Locals(interfaces.REQ_LOCAL_ERROR_KEY)

	if jwtRawFromLocal == nil {
		if localError != nil {
			err, ok := localError.(*interfaces.RequestError)
			if !ok {
				return &interfaces.RequestError{
					StatusCode: 403,
					Code:       interfaces.ERROR_INTERNAL_SERVER,
					Message:    "Cannot Cast Error Token",
					Name:       "NOT_VALID_DECRYPTED_TOKEN",
				}
			}
			return err
		}
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_TOKEN_NOT_PASSED,
			Message:    "Invalid Token or token expired",
			Name:       "ERROR_TOKEN_NOT_PASSED",
		}
	}
	return c.Next()
}

func AllowOnlyValidLoggedInWhatsapp(c *fiber.Ctx) error {
	jwtRawFromLocal := c.Locals(REQ_LOCAL_NUMBER_KEY)
	token, ok := jwtRawFromLocal.(string)
	if !ok {
		return &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "User Token Not Found",
			Name:       "ERROR_INVALID_TOKEN",
		}
	}
	if value, ok := whatsapp_config.WhatsappNumberConfigMap.Tokens[token]; !ok {
		return &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Invalid Token",
			Name:       "ERROR_INVALID_TOKEN",
		}
	} else if value == "" {
		return &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_CONNECTION_LOGGED_OUT,
			Message:    "Connection Not Connected Please Login to Whatsapp Account",
			Name:       "ERROR_CONNECTION_LOGGED_OUT",
		}
	}
	return c.Next()
}
