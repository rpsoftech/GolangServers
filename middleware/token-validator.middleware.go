package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
)

var TokenDecrypterFunctinos func(*fiber.Ctx, *string) error

// fiber middleware for jwt
func TokenDecrypter(c *fiber.Ctx) error {
	reqHeaders := c.GetReqHeaders()
	tokenString, foundToken := reqHeaders[env.RequestTokenHeaderKey]
	if !foundToken || len(tokenString) != 1 || tokenString[0] == "" {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_TOKEN_NOT_BEFORE,
			Message:    "Please Pass Valid Token",
			Name:       "ERROR_TOKEN_NOT_BEFORE",
		})
		return c.Next()
	}
	// userRolesCustomClaim, localErr := .VerifyToken(tokenString[0])
	localErr := TokenDecrypterFunctinos(c, &tokenString[0])
	if localErr != nil {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, localErr)
		return c.Next()
	}
	return c.Next()
}
