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
	telegram_server_apis "github.com/rpsoftech/golang-servers/servers/telegram-server/apis"
	telegram_server_config "github.com/rpsoftech/golang-servers/servers/telegram-server/config"
	telegram_server_middleware "github.com/rpsoftech/golang-servers/servers/telegram-server/middleware"
	telegram_config "github.com/rpsoftech/golang-servers/servers/telegram-server/telegram"
	telegram_server "github.com/rpsoftech/golang-servers/servers/telegram-server/telegram"
	"github.com/rpsoftech/golang-servers/telegram"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

var version string

func main() {
	env.LoadEnv("telegram-server.env")
	println(version)
	go func() {
		os.RemoveAll("./tmp")
		os.Mkdir("./tmp", 0777)
	}()
	outputLogFolderDir := filepath.Join(env.FindAndReturnCurrentDir(), "telegram_server_logs")
	if _, err := utility_functions.Exist(outputLogFolderDir); errors.Is(err, os.ErrNotExist) {
		os.MkdirAll(outputLogFolderDir, 0777)
	}
	telegram_config.OutPutFilePath = ReturnOutPutFilePath(env.FindAndReturnCurrentDir())
	go func() {
		for i, t := range telegram_server_config.TelegramBotConfigMap.Tokens {
			println(i, t)
			bot := telegram.InitAndReturnTelegramBOT(t)
			go bot.UserIdListner()
			telegram_server.ConnectionMap.AddBotConnection(i, telegram_server.CreateTelegramBot(t, bot))
		}
		// telegram_config.ConnectionMap
	}()
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
	app.Static("/swagger", "./swagger")
	telegram_server_apis.AddApis(app.Group("/v1", telegram_server_middleware.TokenDecrypter, telegram_server_middleware.AllowOnlyValidTokenMiddleWare))
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
