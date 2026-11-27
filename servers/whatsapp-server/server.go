package main

import (
	"errors"
	"io"
	"log"

	"github.com/gofiber/fiber/v3"
	whatsapp_logging "github.com/rpsoftech/golang-servers/functions/whatsapp/logging"
	whatsapp_middleware "github.com/rpsoftech/golang-servers/functions/whatsapp/middleware"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_server_apis "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/apis"
)

func setupFiberServer(logStream io.Writer) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 200 * 1024 * 1024,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			var mappedError *interfaces.RequestError
			if !errors.As(err, &mappedError) {
				log.Printf("[Fiber Error] %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(interfaces.RequestError{
					Code:    interfaces.ERROR_INTERNAL_SERVER,
					Message: "Some Internal Error",
					Name:    "Global Error Handler Function",
				})
			}
			return c.Status(mappedError.StatusCode).JSON(mappedError)
		},
	})

	app.Use(whatsapp_logging.NewRedactingLoggerTo(logStream))
	whatsapp_server_apis.AddApis(app.Group("/v1", whatsapp_middleware.TokenDecrypter, whatsapp_middleware.AllowOnlyValidTokenMiddleWare))
	app.Get("/scan/:id", whatsapp_server_apis.OpenBrowserWithQr)
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})

	return app
}
