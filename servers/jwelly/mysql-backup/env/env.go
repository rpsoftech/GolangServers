package mysql_backup_env

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rpsoftech/golang-servers/env"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	mysqldb "github.com/rpsoftech/golang-servers/utility/mysql"
	"github.com/rpsoftech/golang-servers/validator"
)

type SMysqlToSurrealEnv struct {
	Env            *env.DefaultEnvInterface `json:"env" validate:"required"`
	ConfigFileName string                   `json:"configFileName" validate:"required"`
	IsDev          bool
}

type SConfig struct {
	ServerConfig []ServerConfig `json:"serverConfig" validate:"required"`
}

type ServerConfig struct {
	MysqlConfig      mysqldb.MysqldbConfig `json:"mysqlConfig" validate:"required"`
	FileServerType   string                `json:"fileServerType" validate:"required"`
	FileServerConfig map[string]string     `json:"fileServerConfig" validate:"required"`
	Cron             string                `json:"cron" validate:"required"`
	Name             string                `json:"name" validate:"required"`
}

var ServerEnv *SMysqlToSurrealEnv
var ConnectionConfig *SConfig

func init() {
	ServerEnv = &SMysqlToSurrealEnv{
		Env:            env.Env,
		ConfigFileName: env.Env.GetEnv(ConfigFileName),
		IsDev:          env.IsDev,
	}
	if ServerEnv.ConfigFileName == "" {
		ServerEnv.ConfigFileName = "config.json"
	}
	ServerEnv.ConfigFileName = filepath.Join(env.FindAndReturnCurrentDir(), ServerEnv.ConfigFileName)
	if _, err := utility_functions.Exist(ServerEnv.ConfigFileName); err == nil {
		fmt.Printf("Config file exists: %s \n", ServerEnv.ConfigFileName)
	} else if errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("Config file exists: %s \n", ServerEnv.ConfigFileName))
	} else {
		panic("Schrodinger: file may or may not exist. See err for details.")
		// Therefore, do *NOT* use !os.IsNotExist(err) to test for file existence
	}
	config := new(SConfig)
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
