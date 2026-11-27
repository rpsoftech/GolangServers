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

	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/updater"
	"go.mau.fi/whatsmeow/store/sqlstore"
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
		autoConnect(container)
	}

	app := setupFiberServer(os.Stdout)
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
		whatsapp_core.DisconnectAll()
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	whatsapp_core.DisconnectAll()

	log.Println("Server successfully stopped.")
}

// autoConnect starts one connection goroutine per configured token, so a
// token that waits for a QR scan does not block the others.
func autoConnect(container *sqlstore.Container) {
	tokens := make([]string, 0)
	for token := range whatsapp_config.WhatsappNumberConfigMap.TokensSnapshot() {
		tokens = append(tokens, token)
	}
	whatsapp_core.ConnectionMap.ConnectEach(tokens, whatsapp_config.WhatsappNumberConfigMap.GetJID, func(jidString, token string) {
		whatsapp_core.ConnectReserved(jidString, token, container)
	})
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
