package soham_whatsapp_gui_config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/validator"
)

var ServerConfigFilePath = ""
var ServerCmd *exec.Cmd

func init() {
	CurrentDirectory := env.FindAndReturnCurrentDir()
	ServerConfigFilePath = filepath.Join(CurrentDirectory, whatsapp_config.ServerConfigFileName)
}

func ValidateConfig() bool {
	if _, err := utility_functions.Exist(ServerConfigFilePath); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("CONFIG_NOT_EXIST_ON_PATH %s", ServerConfigFilePath))
	}
	_, err := readConfigFileAndReturniserverConfig(ServerConfigFilePath)
	if err != nil {
		return false
		// panic(err)
	}
	return true
}

func readConfigFileAndReturniserverConfig(configFilePath string) (*whatsapp_interfaces.IServerConfig, error) {
	config := new(whatsapp_interfaces.IServerConfig)

	dat, err := os.ReadFile(configFilePath)
	// env.Check(err)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(dat, config)
	// env.Check(err)
	if err != nil {
		return nil, err
	}
	if errs := validator.Validator.Validate(config); len(errs) > 0 {
		panic(fmt.Errorf("CONFIG_ERROR %#v", errs))
	}
	if config.JID == nil {
		config.JID = make(map[string]string)
	}
	config.SetConfigPath(configFilePath)
	return config, nil
}

func SaveConfig(config *whatsapp_interfaces.IServerConfig) {
	byteJson, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return
	}
	err = os.WriteFile(config.GetConfigPath(), byteJson, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
