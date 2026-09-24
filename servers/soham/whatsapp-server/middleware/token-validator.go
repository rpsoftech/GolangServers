package soham_whatsapp_server_middleware

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_services "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/services"
	"github.com/rpsoftech/golang-servers/utility/jwt"
)

type TokenValidator struct {
	accessTokenService *jwt.TokenService
}

var (
	instance *TokenValidator
	once     sync.Once
)

func GetTokenValidator() *TokenValidator {
	once.Do(func() {
		instance = &TokenValidator{
			accessTokenService: soham_whatsapp_server_services.GetAccessTokenService(),
		}
	})
	return instance
}

// fiber middleware for jwt
func (tv *TokenValidator) TokenDecrypter(c fiber.Ctx) error {
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
	claim, err := soham_whatsapp_server_services.ValidateUserRequestToken(tv.accessTokenService, splitToken[1])
	if err != nil {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, err)
		return c.Next()
	}
	c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY, claim.From)
	return c.Next()
}

func (tv *TokenValidator) AllowOnlyValidTokenMiddleWare(c fiber.Ctx) error {
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

func (tv *TokenValidator) AllowOnlyValidLoggedInWhatsapp(c fiber.Ctx) error {
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
	if status, exists := soham_whatsapp_server_services.GetWhatsappService().GetStatus(token); !exists {
		return &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Invalid Token or Session Disconnected",
			Name:       "ERROR_INVALID_TOKEN",
		}
	} else if status != soham_common_req_keys.LOGGED_IN {
		return &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_CONNECTION_LOGGED_OUT,
			Message:    fmt.Sprintf("Connection with number %d is offline (Status: %s)", token, status),
			Name:       "ERROR_CONNECTION_LOGGED_OUT",
			Extra: fiber.Map{
				"status": status,
			},
		}
	}
	return c.Next()
}
