package mysql_backup_interfaces

import (
	env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-mysql/env"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

// "root:rootpasswd@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=true"
// const dsn = "%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true"

type ConfigWithConnection struct {
	ServerConfig *env.ServerConfig
	// ConnectionString  string
	// BaseDir           string
	MysqlHostDbConncetion   *mysqldb.MysqlDBStruct
	MysqlRemoteDbConnection *mysqldb.MysqlDBStruct
}

func ValidateAllConnectionsAndAssign(c *ConfigWithConnection) error {
	if mysqlDb, err := mysqldb.InitalizeMysqlDbWithConfig(&c.ServerConfig.MysqlConfig); err != nil {
		return err
	} else {
		if mysqlRemoteDb, err := mysqldb.InitalizeMysqlDbWithConfig(&c.ServerConfig.MysqlConfig); err != nil {
			mysqlDb.Db.Close()
			return err
		} else {
			c.MysqlRemoteDbConnection = mysqlRemoteDb
		}
		c.MysqlHostDbConncetion = mysqlDb
	}
	return nil
}
