package ecommerce_sync

import (
	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	ecommerce_functions "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/functions"
)

func BasicDetailsSync() {
	ecommerce_functions.SyncSiteTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemUnitTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemGroupTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemMasterTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncStampTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)

}
