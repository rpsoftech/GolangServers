package ecommerce_env

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2/log"
	"github.com/rpsoftech/golang-servers/env"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
)

type (
	IServerMysqlConnection struct {
		Server *mysqldb.MysqlDBStruct
		ERP    *mysqldb.MysqlDBStruct
	}
	IServerConfig struct {
		ServerDatabase         *mysqldb.MysqldbConfig `json:"ServerDatabase" validate:"required"`
		ErpDatabase            *mysqldb.MysqldbConfig `json:"ErpDatabase" validate:"required"`
		AccountTableSyncCron   string                 `json:"AccountTableSyncCron" validate:"required"`
		BasicDetailsSyncCron   string                 `json:"BasicDetailsSyncCron" validate:"required"`
		ItemTagDetailsSyncCron string                 `json:"ItemTagDetailsSyncCron" validate:"required"`
	}

	serverEnv struct {
		DefaultEnv       *env.DefaultEnvInterface
		ACCESS_TOKEN_KEY string `json:"ACCESS_TOKEN_KEY" validate:"required,min=10"`
	}
)

var (
	ServerConfig     *IServerConfig
	MysqlConnections *IServerMysqlConnection
	Env              *serverEnv
)

func Init() *IServerConfig {
	if ServerConfig != nil {
		return ServerConfig
	}
	MysqlConnections = &IServerMysqlConnection{}
	ServerConfig = &IServerConfig{}
	env.LoadEnv("ecommerce-adapter.env")
	log.Debug("ServerEnv Initialized")
	Env = &serverEnv{
		DefaultEnv:       env.Env,
		ACCESS_TOKEN_KEY: env.Env.GetEnv("ACCESS_TOKEN_KEY"),
	}
	env.ValidateEnv(Env)
	configFileName := "server.config.json"
	if e := env.Env.GetEnv("CONFIG_FILE_NAME"); e != "" {
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
