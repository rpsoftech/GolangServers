package mysql_to_surreal_functions

import (
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
	"github.com/rpsoftech/golang-servers/utility/surrealdb"
)

func ValidateAllConnectionsAndAssign(c *ConfigWithConnection) error {
	if surrealDb, err := surrealdb.InitalizeSurrealDbWithConfig(&c.ServerConfig.SurrealConfig); err != nil {
		println("Error in Surreal")
		return err
	} else {
		c.DbConnections.SurrealDbConncetion = surrealDb
	}
	if mysqlDb, err := mysqldb.InitalizeMysqlDbWithConfig(&c.ServerConfig.MysqlConfig); err != nil {
		return err
	} else {
		c.DbConnections.MysqlDbConncetion = mysqlDb
	}
	return nil
}
