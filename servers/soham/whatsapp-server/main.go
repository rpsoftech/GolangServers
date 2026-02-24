package main

import (
	"crypto/tls"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_whatsapp_server_api "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/api"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
	soham_whatsapp_server_middleware "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/middleware"
	soham_whatsapp_server_services "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/services"
	soham_whatsapp_server_websocket "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/websocket"
)

func main() {

	soham_whatsapp_server_services.InialiseWhatsappService(&soham_whatsapp_server_env.ReqestIdMap,
		&soham_whatsapp_server_env.WebsocketConnectionMap)

	app := fiber.New(fiber.Config{
		BodyLimit: 200 * 1024 * 1024,
		ErrorHandler: func(c fiber.Ctx, err error) error {
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
	app.Use("/whatsapp-client", soham_whatsapp_server_middleware.ValidateWhatsAppClientToken)
	app.Get("/whatsapp-client/ws", websocket.New(soham_whatsapp_server_websocket.WhatsappClientWebsocketHandler))
	soham_whatsapp_server_api.AddApis(app.Group("/api"))
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})
	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		hostAndPort = "127.0.0.1"
	}
	hostAndPort = hostAndPort + ":" + env.GetServerPort(env.PORT_KEY)
	app.Listen(hostAndPort, fiber.ListenConfig{
		// TLSConfig: ,
		TLSMinVersion: tls.VersionTLS10,
	})
}
