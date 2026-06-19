package ecommerce_sync

import (
	ecommerce_env "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/env"
	ecommerce_functions "github.com/rpsoftech/golang-servers/servers/jwelly/ecommerce-adapter/functions"
)

func ItemDetailsTagsSync() {
	ecommerce_functions.SyncItemTagTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemTagVariationTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemTagVariationDetailsTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)
	ecommerce_functions.SyncItemTransTable(ecommerce_env.MysqlConnections.Server, ecommerce_env.MysqlConnections.ERP)

}
