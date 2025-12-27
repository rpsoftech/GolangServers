package telegram_server_config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rpsoftech/golang-servers/env"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/validator"
)

var (
	TelegramBotConfigMap *IServerConfig
	CurrentDirectory     string = ""
	serverConfigFilePath string = ""
)

const ServerConfigFileName = "server.config.json"

type IServerConfig struct {
	Tokens map[string]string `json:"tokens" validate:"required"`
}

func (sc *IServerConfig) Save() {
	if serverConfigFilePath == "" {
		serverConfigFilePath = filepath.Join(CurrentDirectory, ServerConfigFileName)
	}
	byteJson, err := json.MarshalIndent(sc, "", "    ")
	if err != nil {
		return
	}
	f, err := os.OpenFile(serverConfigFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Printf("%v \n", err)
		return
	}
	defer f.Close()
	if _, err = f.Write(byteJson); err != nil {
		fmt.Printf("%v \n", err)
	}
}

func ReadConfigFileAndReturnIt(currentDir string) *IServerConfig {
	config := new(IServerConfig)
	configFilePAth := filepath.Join(currentDir, ServerConfigFileName)
	if _, err := utility_functions.Exist(configFilePAth); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("CONFIG_NOT_EXIST_ON_PATH %s", configFilePAth))
	}
	dat, err := os.ReadFile(configFilePAth)
	env.Check(err)
	err = json.Unmarshal(dat, config)
	env.Check(err)
	if errs := validator.Validator.Validate(config); len(errs) > 0 {
		panic(fmt.Errorf("CONFIG_ERROR %#v", errs))
	}
	return config
}
