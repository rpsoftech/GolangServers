package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	"github.com/rpsoftech/golang-servers/interfaces"
	whatsapp_server_apis "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/apis"
	whatsapp_server_middleware "github.com/rpsoftech/golang-servers/servers/whatsapp-server/src/middleware"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/updater"
)

var version = "dev"

const ProjectName = updater.WhatsappProjectName

func main() {
	// 1. Create a root context that listens for OS interrupt signals
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	env.LoadEnv("whatsapp-server.env")
	whatsapp_config.InitaliseWhatsappEnvAndConfig()
	log.Printf("Server Version: %s", version)

	if err := initDirectories(); err != nil {
		log.Fatalf("Directory initialization failed: %v", err)
	}

	// 2. OTA updates: only published builds (updater.BuildEnv set by
	// utility/deploy) update, and only from their own release channel.
	go updater.RunDaemon(ctx, updater.Config{
		Component:      ProjectName,
		CurrentVersion: updater.ParseVersion(version),
	}, cancel)

	whatsapp_core.OutPutFilePath = ReturnOutPutFilePath(env.FindAndReturnCurrentDir())
	container := whatsapp_core.InitSqlContainer()

	if whatsapp_config.Env.AUTO_CONNECT_TO_WHATSAPP {
		go func() {
			for k := range whatsapp_config.WhatsappNumberConfigMap.TokensSnapshot() {
				// Note: Update ConnectToNumber in your whatsapp_core package to accept and respect ctx
				jidString := whatsapp_config.WhatsappNumberConfigMap.GetJID(k)
				whatsapp_core.ConnectToNumber(jidString, k, container)
			}
		}()
	}

	app := setupFiberServer()
	hostAndPort := getHostAndPort()

	// 3. Start server in a non-blocking goroutine
	go func() {
		if err := app.Listen(hostAndPort); err != nil {
			log.Printf("Fiber server stopped: %v", err)
		}
	}()

	// 4. Block until OS signal is received
	<-ctx.Done()
	log.Println("Shutdown signal received. Shutting down gracefully...")

	// 5. Allow up to 10 seconds for existing requests to finish
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server successfully stopped.")
}

func initDirectories() error {
	currentDir := env.FindAndReturnCurrentDir()
	logDir := filepath.Join(currentDir, "whatsapp_server_logs")

	if err := os.RemoveAll("./tmp"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to clear tmp directory: %w", err)
	}
	if err := os.MkdirAll("./tmp", 0755); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}

	if _, err := utility_functions.Exist(logDir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create logs directory: %w", err)
		}
	}
	return nil
}

func setupFiberServer() *fiber.App {
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

	app.Use(logger.New())
	whatsapp_server_apis.AddApis(app.Group("/v1", whatsapp_server_middleware.TokenDecrypter, whatsapp_server_middleware.AllowOnlyValidTokenMiddleWare))
	app.Get("/scan/:id", whatsapp_server_apis.OpenBrowserWithQr)
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})

	return app
}

func getHostAndPort() string {
	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOP {
		hostAndPort = "127.0.0.1"
	}
	return hostAndPort + ":" + env.GetServerPort(env.PORT_KEY)
}

func ReturnOutPutFilePath(currentDir string) string {
	t := time.Now()
	today := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
	return filepath.Join(currentDir, fmt.Sprintf("%d.log.csv", today))
}
