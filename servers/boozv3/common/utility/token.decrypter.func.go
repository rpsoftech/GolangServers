package booz_main_utility

// func TokenDecrypter(c *fiber.Ctx, token *string) error {
// 	userRolesCustomClaim, localErr := jwt.VerifyToken[booz_main_interfaces.GeneralUserAccessRefreshToken](booz_main_services.AccessTokenService, token)
// 	if localErr != nil {
// 		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, localErr)
// 		return nil
// 	}
// 	if e := utility_functions.ValidateStructAndReturnReqError(userRolesCustomClaim, &interfaces.RequestError{
// 		StatusCode: 401,
// 		Code:       interfaces.ERROR_INVALID_TOKEN_SIGNATURE,
// 		Message:    "Invalid Token Structure",
// 		Name:       "ERROR_INVALID_TOKEN_SIGNATURE",
// 	}); e != nil {
// 		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, e)
// 		return nil
// 	}
// 	role := userRolesCustomClaim.Role.String()
// 	if !booz_main_interfaces.ValidateEnumUserRole(role) {

// 		c.Locals(interfaces.REQ_LOCAL_ERROR_KEY, &interfaces.RequestError{
// 			StatusCode: http.StatusBadRequest,
// 			Code:       interfaces.ERROR_TOKEN_ROLE_NOT_FOUND,
// 			Message:    "Invalid Token Role Or Not Found",
// 			Name:       "INVALID_TOKEN_ROLE",
// 		})
// 		return nil
// 	}
// 	c.Locals(booz_main_interfaces.REQ_LOCAL_KEY_ROLE, role)
// 	c.Locals(booz_main_interfaces.REQ_LOCAL_KEY_TOKEN_RAW_DATA, userRolesCustomClaim)
// 	return nil
// }
