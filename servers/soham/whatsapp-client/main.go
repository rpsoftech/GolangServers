package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	whatsapp_logging "github.com/rpsoftech/golang-servers/functions/whatsapp/logging"
	whatsapp_middleware "github.com/rpsoftech/golang-servers/functions/whatsapp/middleware"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_whatsapp_client_apis "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/apis"
	whatsapp_client_core "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/core"
	soham_whatsapp_client_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/env"
	soham_whatsapp_client_websocket "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-client/websocket"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/updater"
)

var version string

func main() {
	soham_whatsapp_client_env.InialiseSohamWhatsappClientEnv()
	log.Printf("🚀 Booting Soham WhatsApp Client - Version: %s", version)

	// 1. Create a cancellable context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 2. Temp directory for web-media downloads. Done before any websocket
	// or connect starts, so no download can race the cleanup.
	if _, err := prepareTmpDir(env.FindAndReturnCurrentDir()); err != nil {
		log.Printf("⚠️ Warning: Failed to prepare tmp directory: %v", err)
	}

	// 3. Auto-Updater. After an update it cancels ctx, so the normal
	// graceful shutdown below runs and main returns with exit code 0; the
	// launcher then restarts the client with the new binary.
	var updateInstalled atomic.Bool
	go func() {
		if !updater.Published() {
			log.Println("[updater] unpublished build: OTA updates disabled")
			return
		}
		updated, err := updater.SelfUpdate(ctx, updater.Config{
			Component:      updater.SohamWhatsappClientProjectName,
			CurrentVersion: updater.ParseVersion(version),
		})
		if err != nil {
			log.Printf("[updater] %v", err)
			return
		}
		if updated {
			log.Println("[updater] update installed, shutting down for restart")
			updateInstalled.Store(true)
			cancel()
		}
	}()

	// 4. Setup Log Directory properly
	outputLogFolderDir := filepath.Join(env.FindAndReturnCurrentDir(), "whatsapp_server_logs")
	if _, err := utility_functions.Exist(outputLogFolderDir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(outputLogFolderDir, 0755); err != nil {
			log.Fatalf("FATAL: Cannot create log directory: %v", err)
		}
	}

	// CRITICAL FIX: Pass the log folder directory, not the root dir
	whatsapp_core.OutPutFilePath = ReturnOutPutFilePath(outputLogFolderDir)

	// 5. Initialize Database Container
	container := whatsapp_core.InitSqlContainer()

	// 6. Connect WhatsApp Numbers & WebSockets
	if whatsapp_config.Env.AUTO_CONNECT_TO_WHATSAPP {
		for k, n := range whatsapp_config.WhatsappNumberConfigMap.TokensSnapshot() {
			uuidToken := k
			numberToken := n
			jidString := whatsapp_config.WhatsappNumberConfigMap.GetJID(uuidToken)
			whatsapp_client_core.SetScanNumber(uuidToken, numberToken)

			go whatsapp_core.ConnectToNumber(jidString, uuidToken, container)

			websocketObj := &soham_whatsapp_client_websocket.WebsocketConnectionObject{
				Url:                   soham_whatsapp_client_env.SocketUrl,
				Conn:                  nil, // Will be established in InitalizeWebsocket
				UUID:                  uuidToken,
				NUMBER:                numberToken,
				WhatsappConnectionMap: whatsapp_core.ConnectionMap, // Note: Consider moving this to a Singleton eventually
			}
			go websocketObj.InitalizeWebsocket(ctx)
			go websocketObj.CheckStatusAndSendResponse()
		}
	}

	// 7. Start Fiber Server with Graceful Shutdown
	app := InitFiberServer()

	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOP {
		hostAndPort = "127.0.0.1"
	}
	hostAndPort = hostAndPort + ":" + env.GetServerPort(env.PORT_KEY)

	go func() {
		log.Printf("🌐 Fiber server listening on %s", hostAndPort)
		if err := app.Listen(hostAndPort); err != nil {
			if ctx.Err() != nil {
				// Shutting down; main decides the exit code.
				return
			}
			log.Fatalf("Fiber server failed: %v", err)
		}
	}()

	// 8. Block main thread until OS termination signal
	<-ctx.Done()
	if updateInstalled.Load() {
		log.Println("🛑 Update installed. Initiating graceful shutdown...")
	} else {
		log.Println("🛑 Termination signal received. Initiating graceful shutdown...")
	}

	// Give active API requests a few seconds to finish before killing the server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("⚠️ Fiber Shutdown Error: %v", err)
	}
	whatsapp_core.DisconnectAll()
	if updateInstalled.Load() {
		log.Println("[updater] exiting for restart")
	}
	log.Println("✅ Client shut down safely.")
}

// errorHandler maps a RequestError, also when wrapped, to its status code
// and JSON body. Any other error is a 500.
func errorHandler(c fiber.Ctx, err error) error {
	var mappedError *interfaces.RequestError
	if !errors.As(err, &mappedError) {
		log.Printf("Internal Server Error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(interfaces.RequestError{
			Code:    interfaces.ERROR_INTERNAL_SERVER,
			Message: "Some Internal Error",
			Name:    "Global Error Handler Function",
		})
	}
	return c.Status(mappedError.StatusCode).JSON(mappedError)
}

func InitFiberServer() *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit:    200 * 1024 * 1024,
		ErrorHandler: errorHandler,
	})

	app.Use(whatsapp_logging.NewRedactingLogger())

	app.Get("/scan/:id", whatsapp_client_core.OpenBrowserWithQr)
	soham_whatsapp_client_apis.AddApis(app.Group("/v1", whatsapp_middleware.TokenDecrypter, whatsapp_middleware.AllowOnlyValidTokenMiddleWare))

	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(interfaces.RequestError{
			Code:    interfaces.ERROR_PATH_NOT_FOUND,
			Message: "Route not found",
			Name:    "ERROR_PATH_NOT_FOUND",
		})
	})

	return app
}

// ReturnOutPutFilePath creates a daily log file path inside the target directory
func ReturnOutPutFilePath(targetDir string) string {
	t := time.Now()
	// Create a unique filename for the day
	today := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
	return filepath.Join(targetDir, fmt.Sprintf("%d.log.csv", today))
}

// prepareTmpDir empties and recreates <baseDir>/tmp, where web-media
// downloads are written, and returns its path.
func prepareTmpDir(baseDir string) (string, error) {
	tmpDir := filepath.Join(baseDir, "tmp")
	if err := os.RemoveAll(tmpDir); err != nil {
		return tmpDir, err
	}
	return tmpDir, os.MkdirAll(tmpDir, 0755)
}
