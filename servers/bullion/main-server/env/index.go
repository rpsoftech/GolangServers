package bullion_main_server_env

import "github.com/rpsoftech/golang-servers/env"

type bullionEnv struct {
	DefaultEnv        *env.DefaultEnvInterface
	ACCESS_TOKEN_KEY  string `json:"ACCESS_TOKEN_KEY" validate:"required,min=100"`
	REFRESH_TOKEN_KEY string `json:"REFRESH_TOKEN_KEY" validate:"required,min=100"`
}

var Env *bullionEnv

func init() {
	env.LoadEnv("bullion-server.env")
	println("Bullion Main ServerEnv Initialized")
	Env = &bullionEnv{
		DefaultEnv:        env.Env,
		ACCESS_TOKEN_KEY:  env.Env.GetEnv("ACCESS_TOKEN_KEY"),
		REFRESH_TOKEN_KEY: env.Env.GetEnv("REFRESH_TOKEN_KEY"),
	}
	env.ValidateEnv(Env)
}
