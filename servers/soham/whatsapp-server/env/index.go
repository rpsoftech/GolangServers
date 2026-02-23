package soham_whatsapp_server_env

import (
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/google/uuid"
	"github.com/rpsoftech/golang-servers/env"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
)

type whatsappEnv struct {
	DefaultEnv *env.DefaultEnvInterface
	BASE_UUID  string    `json:"BASE_UUID" validate:"required"`
	UUIDObj    uuid.UUID `json:"-" validate:"-"`
}

var Env *whatsappEnv

var (
	WebsocketConnectionMap    = make(map[string]*websocket.Conn)
	ConnectionNumberStatusMap = make(map[string]soham_common_req_keys.ConnectionStatus)
)

func init() {
	env.LoadEnv("whatsapp-server.env")
	println("WhatsApp ServerEnv Initialized")
	Env = &whatsappEnv{
		DefaultEnv: env.Env,
		BASE_UUID:  env.Env.GetEnv("BASE_UUID"),
	}
	uuidObj, err := uuid.Parse(Env.BASE_UUID)
	if err != nil {
		panic("Invalid BASE_UUID in env file")
	}
	Env.UUIDObj = uuidObj
	env.ValidateEnv(Env)
}
