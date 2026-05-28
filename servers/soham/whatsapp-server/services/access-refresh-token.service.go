package soham_whatsapp_server_services

import (
	"github.com/gofiber/fiber/v3/log"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
	"github.com/rpsoftech/golang-servers/utility/jwt"
)

var AccessTokenService *jwt.TokenService
var RefreshTokenService *jwt.TokenService

func GetAccessTokenService() *jwt.TokenService {
	if AccessTokenService == nil {
		AccessTokenService = &jwt.TokenService{
			SigningKey: []byte(soham_whatsapp_server_env.Env.ACCESS_TOKEN_KEY),
			Parser:     jwtv5.NewParser(jwtv5.WithValidMethods([]string{"HS256"})),
		}
		log.Info("Token Service Initialized")
	}
	return AccessTokenService
}

func ValidateUserRequestToken(t *jwt.TokenService, token *string) (*soham_common_req_keys.ReqTokenType, error) {
	claimRaw, err := t.Parser.ParseWithClaims(*token, &soham_common_req_keys.ReqTokenType{}, t.Keyfunc)
	if err != nil {
		return nil, err
	}
	claim, ok := claimRaw.Claims.(*soham_common_req_keys.ReqTokenType)
	if !ok {
		err = &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Error InValid Token Body",
			Name:       "ERROR_INVALID_TOKEN_BODY",
			Extra:      err,
		}
		return nil, err
	}
	return claim, nil
}
