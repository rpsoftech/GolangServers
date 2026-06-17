package main

import (
	"log"

	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	ecommerce_functions "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/functions"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

func main() {
	config := ecommerce_env.Init()
	if conn, err := mysqldb.InitializeMysqlDbWithConfig(config.ServerDatabase); err != nil {
		log.Panicf("Connection In Server Database")
		panic(err)
	} else {
		ecommerce_env.MysqlConnections.Server = conn
	}
	if conn, err := mysqldb.InitializeMysqlDbWithConfig(config.ErpDatabase); err != nil {
		log.Panicf("Connection In ERP Database")
		panic(err)
	} else {
		ecommerce_env.MysqlConnections.ERP = conn
	}
	ecommerce_functions.SyncItemUnitTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemGroupTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemMasterTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncStampTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemTagTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemTagVariationTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemTagVariationDetailsTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	log.Printf("DONE")
}
