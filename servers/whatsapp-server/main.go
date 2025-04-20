package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/apis"
	whatsapp_config "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/config"
	"github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/middleware"
	"github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/whatsapp"
)

var version string

// var app *fiber.App

func main() {
	println(version)

	if time.Now().Unix() > 1734512128 {
		println("Please Update The Binary From Keyur Shah")
		println("Press Any Key To Close")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		return
	}
	// args := os.Args
	// if !slices.Contains(args, "--dev") && !slices.Contains(args, "--prod") {
	// 	cmd := exec.Command(filepath.Join(FindAndReturnCurrentDir(), os.Args[0]), "--prod")
	// 	cmd.Stdout = os.Stdout
	// 	err := cmd.Start()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Printf("Just ran subprocess %d, exiting\n", cmd.Process.Pid)
	// 	// time.Sleep(5 * time.Second)
	// 	return
	// }
	go func() {
		os.RemoveAll("./tmp")
		os.Mkdir("./tmp", 0777)
	}()
	outputLogFolderDir := filepath.Join(env.FindAndReturnCurrentDir(), "whatsapp_server_logs")

	if _, err := os.Stat(outputLogFolderDir); errors.Is(err, os.ErrNotExist) {
		os.MkdirAll(outputLogFolderDir, 0777)
	}
	whatsapp.OutPutFilePath = ReturnOutPutFilePath(env.FindAndReturnCurrentDir())
	whatsapp.InitSqlContainer()
	if whatsapp_config.Env.AUTO_CONNECT_TO_WHATSAPP {
		go func() {
			for k := range whatsapp_config.WhatsappNumberConfigMap.Tokens {
				jidString := whatsapp_config.WhatsappNumberConfigMap.JID[k]
				whatsapp.ConnectToNumber(jidString, k)
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
	app.Static("/swagger", "./swagger")
	apis.AddApis(app.Group("/v1", middleware.TokenDecrypter, middleware.AllowOnlyValidTokenMiddleWare))

	app.Get("/scan/:id", apis.OpenBrowserWithQr)
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
