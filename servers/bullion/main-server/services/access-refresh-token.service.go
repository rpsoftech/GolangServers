package bullion_main_server_services

import (
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_env "github.com/rpsoftech/golang-servers/servers/bullion/main-server/env"
	interfacesLocal "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	"github.com/rpsoftech/golang-servers/utility/jwt"
)

var AccessTokenService *jwt.TokenService
var RefreshTokenService *jwt.TokenService

func init() {
	AccessTokenService = &jwt.TokenService{
		SigningKey: []byte(bullion_main_server_env.Env.ACCESS_TOKEN_KEY),
		Parser:     jwtv5.NewParser(jwtv5.WithValidMethods([]string{"HS256"})),
	}
	RefreshTokenService = &jwt.TokenService{
		SigningKey: []byte(bullion_main_server_env.Env.REFRESH_TOKEN_KEY),
		Parser:     jwtv5.NewParser(jwtv5.WithValidMethods([]string{"HS256"})),
	}
	println("Token Service Initialized")
}

func ValidateGeneralUserAccessToken(t *jwt.TokenService, token *string) (*interfacesLocal.GeneralUserAccessRefreshToken, error) {
	claimRaw, err := t.Parser.ParseWithClaims(*token, &interfacesLocal.GeneralUserAccessRefreshToken{}, t.Keyfunc)
	if err != nil {
		return nil, err
	}
	claim, ok := claimRaw.Claims.(*interfacesLocal.GeneralUserAccessRefreshToken)
	if !ok {
		err = &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Error InValid Token Body",
			Name:       "ERROR_INVALID_TOKEN_BODY",
			Extra:      err,
		}
	}
	return claim, err

}
func ValidateGeneralPurposeTokenGeneration(t *jwt.TokenService, token *string) (*interfacesLocal.GeneralPurposeTokenGeneration, error) {
	claimRaw, err := t.Parser.ParseWithClaims(*token, &interfacesLocal.GeneralPurposeTokenGeneration{}, t.Keyfunc)
	if err != nil {
		return nil, err
	}
	claim, ok := claimRaw.Claims.(*interfacesLocal.GeneralPurposeTokenGeneration)
	if !ok {
		err = &interfaces.RequestError{
			StatusCode: 401,
			Code:       interfaces.ERROR_INVALID_TOKEN,
			Message:    "Error InValid Token Body",
			Name:       "ERROR_INVALID_TOKEN_BODY",
			Extra:      err,
		}
	}
	return claim, err
}
