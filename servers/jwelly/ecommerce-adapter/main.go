package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/robfig/cron/v3"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	ecommerce_api "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/internal/api"
	ecommerce_sync "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/sync-func"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

func deferMainFunc() {
	println("Closing...")
	redis.DeferFunction()
}

func main() {
	// Initialize configuration and databases
	config := ecommerce_env.Init()
	InitializeMysqlConnections(config)
	defer deferMainFunc()

	// 1. Initialize & Start Cron Scheduler
	cronTab := cron.New()
	cronTab.AddFunc(ecommerce_env.ServerConfig.AccountTableSyncCron, ecommerce_sync.AccountSync)
	cronTab.AddFunc(ecommerce_env.ServerConfig.BasicDetailsSyncCron, ecommerce_sync.BasicDetailsSync)
	cronTab.AddFunc(ecommerce_env.ServerConfig.ItemTagDetailsSyncCron, ecommerce_sync.ItemDetailsTagsSync)

	log.Println("Starting background cron scheduler...")
	cronTab.Start()

	// 2. Initialize Fiber App
	app := BuildApiServer()
	port := ":" + env.GetServerPort("PORT")
	// 3. Start Fiber Server in a background Goroutine
	// port := ":8080" // Fetch from config if available (e.g., config.Port)
	go func() {
		log.Printf("API Server listening on port %s...\n", port)
		if err := app.Listen(port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fiber server failed to start: %v\n", err)
		}
	}()

	// 4. Graceful Shutdown Listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit // Block main thread until an OS signal is received

	log.Println("Shutdown signal received. Cleaning up...")

	// Stop Cron Jobs gracefully
	cronCtx := cronTab.Stop()
	<-cronCtx.Done()
	log.Println("Cron scheduler stopped.")

	// Shutdown Fiber App with Timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Fiber forced shutdown error: %v\n", err)
	} else {
		log.Println("Fiber API server shut down gracefully.")
	}
}

// InitializeMysqlConnections handles pool initialization
func InitializeMysqlConnections(config *ecommerce_env.IServerConfig) {
	connServer, err := mysqldb.InitializeMysqlDbWithConfig(config.ServerDatabase)
	if err != nil {
		log.Fatalf("Error initializing MySQL Server connection: %v", err)
	}
	ecommerce_env.MysqlConnections.Server = connServer

	connErp, err := mysqldb.InitializeMysqlDbWithConfig(config.ErpDatabase)
	if err != nil {
		log.Fatalf("Error initializing ERP MySQL connection: %v", err)
	}
	ecommerce_env.MysqlConnections.ERP = connErp
}

// BuildApiServer configures and returns the Fiber app instance
func BuildApiServer() *fiber.App {
	app := fiber.New(fiber.Config{
		ServerHeader: "Bullion Server V1.0.0",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			mappedError, ok := err.(*interfaces.RequestError)
			if !ok {
				log.Printf("Unhandled Internal Error: %v\n", err)
				return c.Status(500).JSON(interfaces.RequestError{
					Code:    interfaces.ERROR_INTERNAL_SERVER,
					Message: "Some Internal Error",
					Name:    "Global Error Handler Function",
				})
			}
			if mappedError.LogTheDetails {
				// TODO: Audit log error context
			}
			return c.Status(mappedError.StatusCode).JSON(mappedError)
		},
	})

	app.Use(logger.New())

	// TODO: Attach your API routes here (e.g., app.Get("/api/products", productHandler.GetProducts))
	ecommerce_api.AddApiRoutes(app.Group("/api/v1"))
	return app
}
