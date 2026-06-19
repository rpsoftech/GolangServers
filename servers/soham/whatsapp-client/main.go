package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	whatsapp_middleware "github.com/rpsoftech/golang-servers/functions/whatsapp/middleware"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_whatsapp_client_apis "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/apis"
	whatsapp_client_core "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/core"
	soham_whatsapp_client_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/env"
	soham_whatsapp_client_websocket "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/websoceket"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

var version string

func main() {
	soham_whatsapp_client_env.InialiseSohamWhatsappClientEnv()
	println(version)
	go func() {
		os.RemoveAll("./tmp")
		os.Mkdir("./tmp", 0777)
	}()
	outputLogFolderDir := filepath.Join(env.FindAndReturnCurrentDir(), "whatsapp_server_logs")

	if _, err := utility_functions.Exist(outputLogFolderDir); errors.Is(err, os.ErrNotExist) {
		os.MkdirAll(outputLogFolderDir, 0777)
	}
	// store.DeviceProps.Os = proto.String("BOSS")
	whatsapp_core.OutPutFilePath = ReturnOutPutFilePath(env.FindAndReturnCurrentDir())
	container := whatsapp_core.InitSqlContainer()
	if whatsapp_config.Env.AUTO_CONNECT_TO_WHATSAPP {
		for k, n := range whatsapp_config.WhatsappNumberConfigMap.Tokens {
			jidString := whatsapp_config.WhatsappNumberConfigMap.JID[k]
			whatsapp_config.WhatsappNumberToIDMap[k] = n
			go whatsapp_core.ConnectToNumber(jidString, k, container)
			websokcetObj := soham_whatsapp_client_websocket.WebsocketConnectionObject{
				Url:                   soham_whatsapp_client_env.SocketUrl,
				Conn:                  nil,
				UUID:                  k,
				NUMBER:                n,
				WhatsappConnectionMap: &whatsapp_core.ConnectionMap,
			}
			go websokcetObj.InitalizeWebsocket()
			go websokcetObj.CheckStatusAndSendResponse()
		}
	}

	InitFiberServer()
}

func InitFiberServer() {
	app := fiber.New(fiber.Config{
		BodyLimit: 200 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			mappedError, ok := err.(*interfaces.RequestError)
			if !ok {
				println(err.Error())
				return c.Status(500).JSON(interfaces.RequestError{
					Code:    interfaces.ERROR_INTERNAL_SERVER,
					Message: "Some Internal Error",
					Name:    "Global Error Handler Function",
				})
			}
			return c.Status(mappedError.StatusCode).JSON(mappedError)
		},
	})
	app.Use(logger.New())
	app.Get("/scan/:id", whatsapp_client_core.OpenBrowserWithQr)
	soham_whatsapp_client_apis.AddApis(app.Group("/v1", whatsapp_middleware.TokenDecrypter, whatsapp_middleware.AllowOnlyValidTokenMiddleWare))

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})
	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOP {
		hostAndPort = "127.0.0.1"
	}
	hostAndPort = hostAndPort + ":" + env.GetServerPort(env.PORT_KEY)
	app.Listen(hostAndPort)
}

func ReturnOutPutFilePath(currentDir string) string {
	t := time.Now()
	today := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, t.Nanosecond(), t.Location()).Unix()
	return filepath.Join(currentDir, fmt.Sprintf("%d.log.csv", today))
}
