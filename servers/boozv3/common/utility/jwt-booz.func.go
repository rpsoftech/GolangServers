package booz_main_utility

// func VerifyToken[IClaims jwt.Claims](t *TokenService, token *string) (*IClaims, error) {
// 	claimRaw, err := t.verifyTokenReturnRaw(token)
// 	if err != nil {
// 		return nil, err
// 	}
// 	claim, ok := claimRaw.Claims.(IClaims)
// 	if !ok {
// 		err = &interfaces.RequestError{
// 			StatusCode: 401,
// 			Code:       interfaces.ERROR_INVALID_TOKEN,
// 			Message:    "Error InValid Token Body",
// 			Name:       "ERROR_INVALID_TOKEN_BODY",
// 			Extra:      err,
// 		}
// 	}

// 	return &claim, err
// }
