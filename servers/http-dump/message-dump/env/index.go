package messagedump_env

import "github.com/rpsoftech/golang-servers/env"

type dumpEnv struct {
	DefaultEnv            *env.DefaultEnvInterface
	REDIS_DEFAULT_KEY     string `json:"REDIS_DEFAULT_KEY" validate:"required,min=3"`
	REDIS_DEFAULT_CHANNEL string `json:"REDIS_DEFAULT_CHANNEL" validate:"required,min=3"`
	ServerId              string `json:"SERVER_ID" validate:"required,min=3"`
}

var Env *dumpEnv

func init() {
	env.LoadEnv("message-dump.env")
	println("Dump Server Env Initialized")
	Env = &dumpEnv{
		DefaultEnv:            env.Env,
		REDIS_DEFAULT_KEY:     env.Env.GetEnv("REDIS_DEFAULT_KEY"),
		REDIS_DEFAULT_CHANNEL: env.Env.GetEnv("REDIS_DEFAULT_CHANNEL"),
		ServerId:              env.Env.GetEnv("SERVER_ID"),
		// ACCESS_TOKEN_KEY:  env.Env.GetEnv("ACCESS_TOKEN_KEY"),
		// REFRESH_TOKEN_KEY: env.Env.GetEnv("REFRESH_TOKEN_KEY"),
	}
	env.ValidateEnv(Env)
}
