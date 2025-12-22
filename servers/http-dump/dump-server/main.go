package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	dump_server_apis "github.com/rpsoftech/golang-servers/servers/http-dump/dump-server/api"
	_ "github.com/rpsoftech/golang-servers/servers/http-dump/dump-server/env"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

func deferMainFunc() {
	println("Closing...")
	mongodb.DeferFunction()
	redis.DeferFunction()
}

func main() {
	defer deferMainFunc()
	// env.LoadEnv("http-dump-server.env")
	println("Http Dump Server Env Initialized")
	app := fiber.New(fiber.Config{
		ServerHeader: "Http Dump Server V1.0.0",
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
			if mappedError.LogTheDetails {
				//Todo: Store The Details Of the Error With Body And Other Extra Details Like AUTH KEY AND ETC
			}
			return c.Status(mappedError.StatusCode).JSON(mappedError)
		},
	})
	app.Use(logger.New())
	dump_server_apis.AddApis(app.Group("/v1"))
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})
	// entity := new(dump_server_interfaces.EndPoint)
	// entity.CreateNewEntity("Testing", "TESTING LOCAL", "56397cab-7246-4d1c-ac14-c3446317e67b")
	// entity.BaseEntity.ID = "56397cab-7246-4d1c-ac14-c3446317e67b"
	// dump_server_repos.EndPointRepo.Save(entity)
	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		hostAndPort = "127.0.0.1"
	}
	hostAndPort = hostAndPort + ":" + env.GetServerPort(env.PORT_KEY)
	app.Listen(hostAndPort)
}
