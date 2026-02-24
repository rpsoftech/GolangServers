package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/validator"
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
	tlsConfig := fiber.ListenConfig{
		// TLSConfig: ,
		// TLSMinVersion: tls.VersionTLS10,
	}

	sslPath := filepath.Join(env.FindAndReturnCurrentDir(), "ssl.config.json")
	if _, err := utility_functions.Exist(sslPath); err != nil {
		sslConfig := new(interfaces.SSLConfig)
		dat, err := os.ReadFile(sslPath)
		env.Check(err)
		err = json.Unmarshal(dat, sslConfig)
		env.Check(err)
		if errs := validator.Validator.Validate(sslConfig); len(errs) > 0 {
			panic(fmt.Errorf("SSL_CONFIG_ERROR %#v", errs))
		}
		if _, err := os.Stat(sslConfig.CertFilePath); os.IsNotExist(err) {
			log.Printf("SSL Cert File Not Exist at %s", sslConfig.CertFilePath)
			return
		}
		if _, err := os.Stat(sslConfig.KeyFilePath); os.IsNotExist(err) {
			log.Printf("SSL Key File Not Exist at %s", sslConfig.KeyFilePath)
			return
		}
		cert, err := os.ReadFile(sslConfig.CertFilePath)
		if err != nil {
			log.Printf("Error Reading SSL Cert File at %s, Error: %v", sslConfig.CertFilePath, err)
			return
		}
		key, err := os.ReadFile(sslConfig.KeyFilePath)
		if err != nil {
			log.Printf("Error Reading SSL Key File at %s, Error: %v", sslConfig.KeyFilePath, err)
			return
		}
		certificate, err := tls.X509KeyPair(cert, key)
		if err != nil {
			log.Printf("Error Loading SSL Certificate, Error: %v", err)
			return
		}
		tlsConfig.TLSConfig = &tls.Config{
			GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return &certificate, nil
			},
			// Certificates: []tls.Certificate{certificate},
		}
		// tlsConfig.TLSConfig.Certificates =
		// key := sslConfig.KeyFilePath
	} else {
		log.Printf("SSL File Path Not Exist at %s", sslPath)
	}
	app.Listen(hostAndPort, tlsConfig)
}
