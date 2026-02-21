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
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_client_core "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/core"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/utility/whatsapp"
	whatsapp_config "github.com/rpsoftech/golang-servers/utility/whatsapp/config"
)

var version string

// var app *fiber.App

func main() {
	env.LoadEnv("whatsapp-client.env")
	println(version)
	go func() {
		os.RemoveAll("./tmp")
		os.Mkdir("./tmp", 0777)
	}()
	outputLogFolderDir := filepath.Join(env.FindAndReturnCurrentDir(), "whatsapp_server_logs")

	if _, err := utility_functions.Exist(outputLogFolderDir); errors.Is(err, os.ErrNotExist) {
		os.MkdirAll(outputLogFolderDir, 0777)
	}
	whatsapp_interfaces.OutPutFilePath = ReturnOutPutFilePath(env.FindAndReturnCurrentDir())
	container := whatsapp_interfaces.InitSqlContainer()
	if whatsapp_config.Env.AUTO_CONNECT_TO_WHATSAPP {
		go func() {
			for k := range whatsapp_config.WhatsappNumberConfigMap.Tokens {
				jidString := whatsapp_config.WhatsappNumberConfigMap.JID[k]
				whatsapp_interfaces.ConnectToNumber(jidString, k, container)
			}
		}()
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
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})
	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
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
