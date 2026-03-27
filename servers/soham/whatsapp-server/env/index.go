package soham_whatsapp_server_env

import (
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3/log"
	"github.com/google/uuid"
	"github.com/rpsoftech/golang-servers/env"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
)

type (
	whatsappEnv struct {
		DefaultEnv       *env.DefaultEnvInterface
		BASE_UUID        string    `json:"BASE_UUID" validate:"required"`
		ACCESS_TOKEN_KEY string    `json:"ACCESS_TOKEN_KEY" validate:"required,min=10"`
		UUIDObj          uuid.UUID `json:"-" validate:"-"`
	}
	WebsocketConnection struct {
		Conn   *websocket.Conn
		Status soham_common_req_keys.ConnectionStatus
	}
)

var Env *whatsappEnv

var (
	WebsocketConnectionMap    = make(map[int]*WebsocketConnection)
	ConnectionNumberStatusMap = make(map[int]soham_common_req_keys.ConnectionStatus)
	ReqestIdMap               = make(map[string]chan any)
)

func init() {
	env.LoadEnv("whatsapp-server.env")
	log.Debug("WhatsApp ServerEnv Initialized")
	Env = &whatsappEnv{
		DefaultEnv:       env.Env,
		BASE_UUID:        env.Env.GetEnv("BASE_UUID"),
		ACCESS_TOKEN_KEY: env.Env.GetEnv("ACCESS_TOKEN_KEY"),
	}
	uuidObj, err := uuid.Parse(Env.BASE_UUID)
	if err != nil {
		panic("Invalid BASE_UUID in env file")
	}
	Env.UUIDObj = uuidObj
	env.ValidateEnv(Env)
}

func (c *WebsocketConnection) SendMessage(reqid string, messageType soham_common_req_keys.MessageType, s any) error {
	return c.Conn.WriteJSON(soham_common_req_keys.WhatsappClientMessage{
		ReqId:   reqid,
		Type:    messageType,
		Message: s,
	})
}
