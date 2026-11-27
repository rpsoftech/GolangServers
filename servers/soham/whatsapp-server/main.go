package main

import (
	"fmt"
	"path/filepath"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rpsoftech/golang-servers/env"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_api "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/api"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
	soham_whatsapp_server_middleware "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/middleware"
	soham_whatsapp_server_websocket "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/websocket"
)

func main() {
	soham_whatsapp_server_env.Init()
	app := fiber.New(fiber.Config{
		BodyLimit:    bodyLimit,
		ErrorHandler: errorHandler,
	})
	// A panic in a handler must not take the whole server down.
	app.Use(recover.New())
	app.Use(logger.New(
		logger.Config{
			Format: "${time} | ${status} | ${latency} | ${number} | ${ip} | ${path} | ${error}\n",
			CustomTags: map[string]logger.LogFunc{
				"number": func(output logger.Buffer, c fiber.Ctx, data *logger.Data, extraParam string) (int, error) {
					val := c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY)
					if val == nil {
						return output.WriteString("N/A")
					}
					return fmt.Fprintf(output, "%v", val)
				},
			},
		},
	))
	app.Use("/whatsapp-client", soham_whatsapp_server_middleware.ValidateWhatsAppClientToken)
	app.Get("/whatsapp-client/ws", websocket.New(soham_whatsapp_server_websocket.GetWhatsappClientWebsocketHandler().Handle))
	soham_whatsapp_server_api.AddApis(app.Group("/api"))
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})
	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOP {
		hostAndPort = "127.0.0.1"
	}
	hostAndPort = hostAndPort + ":" + env.GetServerPort(env.PORT_KEY)
	listenConfig := fiber.ListenConfig{}
	sslPath := filepath.Join(env.FindAndReturnCurrentDir(), "ssl.config.json")
	tlsConfig, err := loadTLSConfig(sslPath)
	if err != nil {
		log.Fatalf("SSL setup failed: %v", err)
	}
	if tlsConfig != nil {
		listenConfig.TLSConfig = tlsConfig
	} else {
		log.Infof("SSL config not found at %s, serving without TLS", sslPath)
	}
	if err := app.Listen(hostAndPort, listenConfig); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
