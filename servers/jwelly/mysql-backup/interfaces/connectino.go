package mysql_backup_interfaces

import (
	"fmt"
	"os"
	"path/filepath"

	coreEnv "github.com/rpsoftech/golang-servers/env"
	env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup/env"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

// "root:rootpasswd@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=true"
const dsn = "%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true"

type ConfigWithConnection struct {
	ServerConfig      *env.ServerConfig
	ConnectionString  string
	BaseDir           string
	MysqlDbConncetion *mysqldb.MysqlDBStruct
	SFileServerConfig IFileServerConfigInterface
}

func ValidateAllConnectionsAndAssign(c *ConfigWithConnection) error {
	if mysqlDb, err := mysqldb.InitalizeMysqlDbWithConfig(&c.ServerConfig.MysqlConfig); err != nil {
		return err
	} else {
		// mysqlDb.Db.Close()
		c.BaseDir = filepath.Join(coreEnv.FindAndReturnCurrentDir(), c.ServerConfig.Name)
		if exists, err := utility_functions.Exist(c.BaseDir); !exists && err == nil {
			os.MkdirAll(c.BaseDir, 0777)
		} else if err != nil {
			println(err.Error())
		}
		c.ConnectionString = fmt.Sprintf(dsn, c.ServerConfig.MysqlConfig.MYSQL_USERNAME, c.ServerConfig.MysqlConfig.MYSQL_PASSWORD, c.ServerConfig.MysqlConfig.MYSQL_HOST, c.ServerConfig.MysqlConfig.MYSQL_PORT, c.ServerConfig.MysqlConfig.MYSQL_DATABASE)
		c.MysqlDbConncetion = mysqlDb
		if c.ServerConfig.FileServerType == "TYPE1" {
			c.SFileServerConfig = &SFileServerType1{
				FolderPath: c.ServerConfig.FileServerConfig["folderPath"],
				SFileServerConfig: &SFileServerConfig{
					URL:   c.ServerConfig.FileServerConfig["url"],
					TOKEN: c.ServerConfig.FileServerConfig["token"],
				},
			}
			if _, err := c.SFileServerConfig.Validate(); err != nil {
				panic(err)
			}
		}
	}
	return nil
}
