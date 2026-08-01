package ecommerce_env

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

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
	once             sync.Once
)

// Init initializes the environment configuration as a thread-safe singleton.
func Init() *IServerConfig {
	once.Do(func() {
		MysqlConnections = &IServerMysqlConnection{}
		ServerConfig = &IServerConfig{}

		env.LoadEnv("ecommerce-adapter.env")
		log.Println("[INFO] ServerEnv Initialized")

		Env = &serverEnv{
			DefaultEnv:       env.Env,
			ACCESS_TOKEN_KEY: env.Env.GetEnv("ACCESS_TOKEN_KEY"),
		}
		env.ValidateEnv(Env)

		configFileName := "server.config.json"
		if e := env.Env.GetEnv("CONFIG_FILE_NAME"); e != "" {
			configFileName = e
		}

		configPath := filepath.Join(env.FindAndReturnCurrentDir(), configFileName)
		exists, err := utility_functions.Exist(configPath)
		if !exists || err != nil {
			log.Fatalf("[FATAL] Config file not found at path %s: %v", configPath, err)
		}

		dat, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatalf("[FATAL] Failed to read config file at %s: %v", configPath, err)
		}

		if err := json.Unmarshal(dat, ServerConfig); err != nil {
			log.Fatalf("[FATAL] Failed to unmarshal config JSON from %s: %v", configPath, err)
		}
	})

	return ServerConfig
}
