package bullion_main_server_middleware

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

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
	userRolesCustomClaim, localErr := services.AccessTokenService.VerifyToken(tokenString[0])
	if localErr != nil {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, localErr)
		return c.Next()
	}
	if e := utility_functions.ValidateStructAndReturnReqError(userRolesCustomClaim, &interfaces.RequestError{
		StatusCode: 401,
		Code:       interfaces.ERROR_INVALID_TOKEN_SIGNATURE,
		Message:    "Invalid Token Structure",
		Name:       "ERROR_INVALID_TOKEN_SIGNATURE",
	}); e != nil {
		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, e)
		return c.Next()
	}
	role := userRolesCustomClaim.Role.String()
	if !bullion_main_server_interfaces.ValidateEnumUserRole(role) {

		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, &interfaces.RequestError{
			StatusCode: http.StatusBadRequest,
			Code:       interfaces.ERROR_TOKEN_ROLE_NOT_FOUND,
			Message:    "Invalid Token Role Or Not Found",
			Name:       "INVALID_TOKEN_ROLE",
		})
		return c.Next()
	}

	c.Locals(interfaces.REQ_LOCAL_KEY_ROLE, role)
	c.Locals(interfaces.REQ_LOCAL_KEY_TOKEN_RAW_DATA, userRolesCustomClaim)
	return c.Next()
}

func AllowOnlyValidTokenMiddleWare(c *fiber.Ctx) error {
	jwtRawFromLocal := c.Locals(interfaces.REQ_LOCAL_KEY_TOKEN_RAW_DATA)
	localError := c.Locals(interfaces.REQ_LOCAL_ERROR_KEY)

	if jwtRawFromLocal == nil {
		if localError != nil {
			err, ok := localError.(*interfaces.RequestError)
			if !ok {
				return &interfaces.RequestError{
					StatusCode: 403,
					Code:       interfaces.ERROR_TOKEN_EXPIRED,
					Message:    "Cannot Cast Error Token",
					Name:       "NOT_VALID_DECRYPTED_TOKEN",
				}
			}
			return err
		}
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_TOKEN_EXPIRED,
			Message:    "Invalid Token or token expired",
			Name:       "NOT_VALID_DECRYPTED_TOKEN",
		}
	}

	jwtRaw, ok := jwtRawFromLocal.(*jwt.GeneralUserAccessRefreshToken)
	if !ok {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_TOKEN_EXPIRED,
			Message:    "Invalid Token or token expired",
			Name:       "NOT_VALID_DECRYPTED_TOKEN",
		}
	}
	if jwtRaw.BullionId == "" {
		return &interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Invalid Bullion Id in Token",
			Name:       "INVALID_BULLION_ID_IN_TOKEN",
		}
	}

	c.Locals(interfaces.REQ_LOCAL_UserID, jwtRaw.UserId)
	c.Locals(interfaces.REQ_LOCAL_BullionId_KEY, jwtRaw.BullionId)
	return c.Next()
}
