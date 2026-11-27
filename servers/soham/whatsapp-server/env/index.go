package soham_whatsapp_server_env

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	fastws "github.com/fasthttp/websocket"
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
	// WSConn is the part of a websocket connection the server uses. It is an
	// interface so tests can replace the real connection.
	WSConn interface {
		WriteMessage(messageType int, data []byte) error
		WriteControl(messageType int, data []byte, deadline time.Time) error
		SetWriteDeadline(t time.Time) error
		Close() error
	}
	// WebsocketConnection is one connected WhatsApp client. Writes are
	// serialized because a websocket connection supports only one concurrent
	// writer.
	WebsocketConnection struct {
		Conn    WSConn
		writeMu sync.Mutex
	}
)

// WriteWait bounds a single websocket write, so a client that stopped reading
// cannot block senders forever. It is a variable so tests can shorten it.
var WriteWait = 2 * time.Minute

// MaxMessageSize is the largest message SendMessage writes. The client
// closes the connection on a larger message. It is a variable so tests can
// shorten it.
var MaxMessageSize = soham_common_req_keys.MaxWebsocketMessageSize

// ErrMessageTooLarge is returned by SendMessage for a message over
// MaxMessageSize. Nothing is written, so the connection stays usable.
var ErrMessageTooLarge = errors.New("message is larger than the websocket message limit")

var Env *whatsappEnv

// Init loads whatsapp-server.env and validates it. main calls it before
// anything reads Env.
func Init() {
	env.LoadEnv(envFilePath())
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

// SendMessage writes one message to the client. It is safe for concurrent use.
// The message is encoded once, outside the write lock, and a message over
// MaxMessageSize is rejected with ErrMessageTooLarge before anything is
// written.
func (c *WebsocketConnection) SendMessage(reqid string, messageType soham_common_req_keys.MessageType, s any) error {
	data, err := json.Marshal(soham_common_req_keys.WhatsappClientMessage{
		ReqId:   reqid,
		Type:    messageType,
		Message: s,
	})
	if err != nil {
		return err
	}
	if len(data) > MaxMessageSize {
		return fmt.Errorf("%w: %d bytes, limit %d", ErrMessageTooLarge, len(data), MaxMessageSize)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := c.Conn.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
		return err
	}
	return c.Conn.WriteMessage(fastws.TextMessage, data)
}

// envFileName is the env file next to the executable.
const envFileName = "whatsapp-server.env"

// envFilePath returns the env file in the executable's directory, like the
// client does, and falls back to the working directory (go run, tests).
func envFilePath() string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), envFileName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return envFileName
}
