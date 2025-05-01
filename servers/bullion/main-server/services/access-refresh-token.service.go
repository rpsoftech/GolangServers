package bullion_main_server_services

import (
	bullion_main_server_env "github.com/rpsoftech/golang-servers/servers/bullion/main-server/env"
	"github.com/rpsoftech/golang-servers/utility/jwt"
)

var AccessTokenService *jwt.TokenService
var RefreshTokenService *jwt.TokenService

func init() {
	AccessTokenService = &jwt.TokenService{
		SigningKey: []byte(bullion_main_server_env.Env.ACCESS_TOKEN_KEY),
	}
	RefreshTokenService = &jwt.TokenService{
		SigningKey: []byte(bullion_main_server_env.Env.REFRESH_TOKEN_KEY),
	}
	println("Token Service Initialized")
}
