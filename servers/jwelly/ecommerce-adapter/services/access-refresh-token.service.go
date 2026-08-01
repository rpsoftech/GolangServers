package ecommerce_services

import (
	"sync"

	"github.com/gofiber/fiber/v3/log"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/rpsoftech/golang-servers/interfaces"
	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	"github.com/rpsoftech/golang-servers/utility/jwt"
)

var (
	accessTokenService *jwt.TokenService
	// refreshTokenService *jwt.TokenService
	onceAccessToken sync.Once
	// onceRefreshToken    sync.Once
)

// GetAccessTokenService returns a thread-safe singleton instance of the access token service
func GetAccessTokenService() *jwt.TokenService {
	onceAccessToken.Do(func() {
		accessTokenService = &jwt.TokenService{
			SigningKey: []byte(ecommerce_env.Env.ACCESS_TOKEN_KEY),
			Parser:     jwtv5.NewParser(jwtv5.WithValidMethods([]string{"HS256"})),
		}
		log.Info("Access Token Service Initialized")
	})
	return accessTokenService
}

// func GetRefreshTokenService() *jwt.TokenService {
// 	onceRefreshToken.Do(func() {
// 		refreshTokenService = &jwt.TokenService{
// 			SigningKey: []byte(ecommerce_env.Env),
// 			Parser:     jwtv5.NewParser(jwtv5.WithValidMethods([]string{"HS256"})),
// 		}
// 		log.Info("Access Token Service Initialized")
// 	})
// 	return refreshTokenService
// }

// ValidateUserRequestToken parses and validates incoming user tokens securely
func ValidateUserRequestToken(t *jwt.TokenService, token *string) (*soham_common_req_keys.ReqTokenType, error) {
	claimRaw, err := t.Parser.ParseWithClaims(*token, &soham_common_req_keys.ReqTokenType{}, t.Keyfunc)
	if err != nil {
		return nil, err
	}

	claim, ok := claimRaw.Claims.(*soham_common_req_keys.ReqTokenType)
	if !ok {
		reqErr := &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Error InValid Token Body",
			Name:       "ERROR_INVALID_TOKEN_BODY",
			Extra:      err,
		}
		return nil, reqErr
	}

	return claim, nil
}
