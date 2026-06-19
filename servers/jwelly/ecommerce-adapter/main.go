package main

import (
	"log"

	"github.com/robfig/cron/v3"
	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	ecommerce_sync "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/sync-func"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

// main function is the entry point of the program
func main() {
	// Initialize the MySQL connections
	InitaliseMysqlConnections()

	// Create a new cron scheduler
	cronTab := cron.New()

	// Add functions to the cron scheduler
	// Account Sync function will be executed every 5 minutes
	cronTab.AddFunc(ecommerce_env.ServerConfig.AccountTableSyncCron, ecommerce_sync.AccountSync)
	// Basic Details Sync function will be executed every 10 minutes
	cronTab.AddFunc(ecommerce_env.ServerConfig.BasicDetailsSyncCron, ecommerce_sync.BasicDetailsSync)
	// Item Details and Tags Sync function will be executed every 15 minutes
	cronTab.AddFunc(ecommerce_env.ServerConfig.ItemTagDetailsSyncCron, ecommerce_sync.ItemDetailsTagsSync)

	// Start the cron scheduler
	log.Printf("Starting the cron scheduler...")
	cronTab.Start()

	// Log a message when the cron scheduler is done starting
	log.Printf("DONE")
}

// Initialize the MySQL connections
func InitaliseMysqlConnections() {
	config := ecommerce_env.Init()

	if conn, err := mysqldb.InitializeMysqlDbWithConfig(config.ServerDatabase); err != nil {
		log.Fatal("Error initializing MySQL connection:", err)
		panic(err)
	} else {
		ecommerce_env.MysqlConnections.Server = conn
	}

	if conn, err := mysqldb.InitializeMysqlDbWithConfig(config.ErpDatabase); err != nil {
		log.Fatal("Error initializing ERP MySQL connection:", err)
		panic(err)
	} else {
		ecommerce_env.MysqlConnections.ERP = conn
	}
}
