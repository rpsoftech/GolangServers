package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
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
					Extra:   err,
				})
			}
			return c.Status(mappedError.StatusCode).JSON(mappedError)
		},
	})
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
	app.Get("/whatsapp-client/ws", websocket.New(soham_whatsapp_server_websocket.WhatsappClientWebsocketHandler))
	soham_whatsapp_server_api.AddApis(app.Group("/api"))
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})
	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOP {
		hostAndPort = "127.0.0.1"
	}
	hostAndPort = hostAndPort + ":" + env.GetServerPort(env.PORT_KEY)
	tlsConfig := fiber.ListenConfig{
		// TLSConfig: ,
		// TLSMinVersion: tls.VersionTLS10,
	}

	sslPath := filepath.Join(env.FindAndReturnCurrentDir(), "ssl.config.json")
	if _, err := utility_functions.Exist(sslPath); err == nil {
		sslConfig := new(interfaces.SSLConfig)
		dat, err := os.ReadFile(sslPath)
		env.Check(err)
		err = json.Unmarshal(dat, sslConfig)
		env.Check(err)
		if errs := validator.Validator.Validate(sslConfig); len(errs) > 0 {
			panic(fmt.Errorf("SSL_CONFIG_ERROR %#v", errs))
		}
		if exist, _ := utility_functions.Exist(sslConfig.CertFilePath); !exist {
			log.Errorf("SSL Cert File Not Exist at %s", sslConfig.CertFilePath)
			return
		}
		if exist, _ := utility_functions.Exist(sslConfig.KeyFilePath); !exist {
			log.Errorf("SSL Key File Not Exist at %s", sslConfig.KeyFilePath)
			return
		}
		certificate, err := tls.LoadX509KeyPair(sslConfig.CertFilePath, sslConfig.KeyFilePath)
		if err != nil {
			log.Errorf("Error Loading SSL Certificate, Error: %v", err)
			return
		}
		caCertPool := x509.NewCertPool()
		caCert, err := os.ReadFile("/Users/keyurshah/Projects/GolangServers/ssl/fullchain.crt")
		if err != nil {
			log.Errorf("Error Reading CA Certificate File, Error: %v", err)
			return
		}
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.CertClientFile = sslConfig.CertFilePath
		tlsConfig.TLSConfig = &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{certificate},
		}
		// tlsConfig.TLSConfig.Certificates =
		// key := sslConfig.KeyFilePath
	} else {
		log.Infof("SSL File Path Not Exist at %s", sslPath)
	}
	app.Listen(hostAndPort, tlsConfig)
}
