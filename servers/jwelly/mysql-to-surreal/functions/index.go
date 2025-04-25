package mysql_to_surreal_functions

import (
	mysql_to_surreal_env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/env"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
	"github.com/rpsoftech/golang-servers/utility/surrealdb"
)

type ConfigWithConnection struct {
	ServerConfig  *mysql_to_surreal_env.ServerConfig
	DbConnections *ConnectionsToDb
}

type ConnectionsToDb struct {
	MysqlDbConncetion   *mysqldb.MysqlDBStruct
	SurrealDbConncetion *surrealdb.SurrealDBStruct
}
