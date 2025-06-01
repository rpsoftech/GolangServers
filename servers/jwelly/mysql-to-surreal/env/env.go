package mysql_to_surreal_env

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rpsoftech/golang-servers/env"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
	"github.com/rpsoftech/golang-servers/utility/surrealdb"
	"github.com/rpsoftech/golang-servers/validator"
)

type MysqlToSurrealEnv struct {
	Env             *env.DefaultEnvInterface `json:"env" validate:"required"`
	ConfigFileName  string                   `json:"configFileName" validate:"required"`
	RefreshDatabase bool                     `json:"refreshDatabase"`
	IsDev           bool
}

type IConfig struct {
	ServerConfig []ServerConfig `json:"serverConfig" validate:"required"`
}

type ServerConfig struct {
	MysqlConfig   mysqldb.MysqldbConfig     `json:"mysqlConfig" validate:"required"`
	SurrealConfig surrealdb.SurrealdbConfig `json:"surrealConfig" validate:"required"`
	Cron          string                    `json:"cron" validate:"required"`
	Name          string                    `json:"name" validate:"required"`
}

var ServerEnv *MysqlToSurrealEnv
var ConnectionConfig *IConfig

func init() {
	refreshDatabaseString := env.Env.GetEnv(RefreshDatabase)
	refreshDatabase := false
	if refreshDatabaseString == "true" {
		refreshDatabase = true
	}
	ServerEnv = &MysqlToSurrealEnv{
		Env:             env.Env,
		ConfigFileName:  env.Env.GetEnv(ConfigFileName),
		RefreshDatabase: refreshDatabase,
		IsDev:           env.IsDev,
	}
	if ServerEnv.ConfigFileName == "" {
		ServerEnv.ConfigFileName = "config.json"
	}
	ServerEnv.ConfigFileName = filepath.Join(env.FindAndReturnCurrentDir(), ServerEnv.ConfigFileName)
	if _, err := os.Stat(ServerEnv.ConfigFileName); err == nil {
		fmt.Printf("Config file exists: %s \n", ServerEnv.ConfigFileName)
	} else if errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("Config file exists: %s \n", ServerEnv.ConfigFileName))
	} else {
		panic("Schrodinger: file may or may not exist. See err for details.")
		// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
	}
	config := new(IConfig)
	dat, err := os.ReadFile(ServerEnv.ConfigFileName)
	env.Check(err)
	err = json.Unmarshal(dat, config)
	env.Check(err)
	if errs := validator.Validator.Validate(config); len(errs) > 0 {
		panic(fmt.Errorf("CONFIG_ERROR %#v", errs))
	}
	ConnectionConfig = config
	errs := validator.Validator.Validate(ServerEnv)
	if len(errs) > 0 {
		errsJson, _ := json.Marshal(errs)
		fmt.Printf("%s", errsJson)
		panic(errs[0])
	}
}
