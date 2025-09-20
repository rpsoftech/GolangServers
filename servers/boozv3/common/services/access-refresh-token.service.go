package booz_main_services

import (
	booz_main_env "github.com/rpsoftech/golang-servers/servers/boozv3/common/env"
	"github.com/rpsoftech/golang-servers/utility/jwt"
)

var AccessTokenService *jwt.TokenService
var RefreshTokenService *jwt.TokenService

func init() {
	AccessTokenService = &jwt.TokenService{
		SigningKey: []byte(booz_main_env.Env.ACCESS_TOKEN_KEY),
	}
	RefreshTokenService = &jwt.TokenService{
		SigningKey: []byte(booz_main_env.Env.REFRESH_TOKEN_KEY),
	}
	println("Token Service Initialized")
}
