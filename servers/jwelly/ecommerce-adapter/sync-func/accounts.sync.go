package ecommerce_sync

import (
	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	ecommerce_functions "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/functions"
)

func AccountSync() {
	ecommerce_functions.SyncAccountGroupTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncAccountMasterTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
}
