package soham_whatsapp_server_middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
	soham_whatsapp_server_services "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/services"
	"github.com/rpsoftech/golang-servers/utility/jwt"
)

var accessTokenService *jwt.TokenService

func init() {
	accessTokenService = soham_whatsapp_server_services.GetAccessTokenService()
}

// fiber middleware for jwt
func TokenDecrypter(c fiber.Ctx) error {
	// reqHeaders :=
	tokenString := c.Get("Authorization")
	if tokenString == "" {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_TOKEN_NOT_BEFORE,
			Message:    "Please Pass Valid Token",
			Name:       "ERROR_TOKEN_NOT_PASSED",
		})
		return c.Next()
	}
	splitToken := strings.Split(tokenString, " ")
	if len(splitToken) != 2 {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_TOKEN_NOT_BEFORE,
			Message:    "Please Pass Valid Token",
			Name:       "ERROR_TOKEN_NOT_PASSED",
		})
		return c.Next()
	}
	claim, err := soham_whatsapp_server_services.ValidateUserRequestToken(accessTokenService, &splitToken[1])
	if err != nil {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, err)
		return c.Next()
	}
	c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY, claim.From)
	return c.Next()
}

func AllowOnlyValidTokenMiddleWare(c fiber.Ctx) error {
	jwtRawFromLocal := c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY)
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

func AllowOnlyValidLoggedInWhatsapp(c fiber.Ctx) error {
	jwtRawFromLocal := c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY)
	token, ok := jwtRawFromLocal.(int)
	if !ok {
		return &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "User Token Not Found",
			Name:       "ERROR_INVALID_TOKEN",
		}
	}
	if value, ok := soham_whatsapp_server_env.ConnectionNumberStatusMap[token]; !ok {
		return &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Invalid Token",
			Name:       "ERROR_INVALID_TOKEN",
		}
	} else if value != soham_common_req_keys.LOGGED_IN {
		return &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_CONNECTION_LOGGED_OUT,
			Message:    fmt.Sprintf("Connection with number %d is not Loggedin Or Connection offline status is %d", token, value),
			Name:       "ERROR_CONNECTION_LOGGED_OUT",
			Extra: fiber.Map{
				"status": value,
			},
		}
	}
	return c.Next()
}
