package ecommerce_env

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rpsoftech/golang-servers/env"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

type IServerMysqlConnection struct {
	Server *mysqldb.MysqlDBStruct
	ERP    *mysqldb.MysqlDBStruct
}
type IServverConfig struct {
	ServerDatabase         *mysqldb.MysqldbConfig `json:"ServerDatabase" validate:"required"`
	ErpDatabase            *mysqldb.MysqldbConfig `json:"ErpDatabase" validate:"required"`
	AccountTableSyncCron   string                 `json:"AccountTableSyncCron" validate:"required"`
	BasicDetailsSyncCron   string                 `json:"BasicDetailsSyncCron" validate:"required"`
	ItemTagDetailsSyncCron string                 `json:"ItemTagDetailsSyncCron" validate:"required"`
}

var ServerConfig *IServverConfig
var MysqlConnections *IServerMysqlConnection

// var
func Init() *IServverConfig {
	if ServerConfig != nil {
		return ServerConfig
	}
	MysqlConnections = &IServerMysqlConnection{}
	ServerConfig = &IServverConfig{}
	env.LoadEnv("ecommerce-adapter.env")
	configFileName := "server.config.json"
	if e := env.Env.GetEnv("CONFGI_FILE_NAME"); e != "" {
		configFileName = e
	}
	configFileName = filepath.Join(env.FindAndReturnCurrentDir(), configFileName)
	if e, err := utility_functions.Exist(configFileName); !e || err != nil {
		panic(err)
	}
	dat, err := os.ReadFile(configFileName)
	env.Check(err)
	err = json.Unmarshal(dat, ServerConfig)
	env.Check(err)
	return ServerConfig
}
