package whatsapp_interfaces

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/validator"
)

type (
	EnvInterface struct {
		ALLOW_LOCAL_NO_AUTH      bool `json:"ALLOW_LOCAL_NO_AUTH" validate:"boolean"`
		AUTO_CONNECT_TO_WHATSAPP bool `json:"AUTO_CONNECT_TO_WHATSAPP" validate:"boolean"`
		OPEN_BROWSER_FOR_SCAN    bool `json:"OPEN_BROWSER_FOR_SCAN" validate:"boolean"`
	}
	IServerConfig struct {
		Tokens         map[string]string `json:"tokens" validate:"required"`
		JID            map[string]string `json:"JID"`
		configFilePath string            `json:"-"`
	}
)

func ReadConfigFileAndReturniserverConfig(configFilePath string) *IServerConfig {
	config := new(IServerConfig)

	dat, err := os.ReadFile(configFilePath)
	env.Check(err)
	err = json.Unmarshal(dat, config)
	// env.Check(err)
	if errs := validator.Validator.Validate(config); len(errs) > 0 {
		panic(fmt.Errorf("CONFIG_ERROR %#v", errs))
	}
	if config.JID == nil {
		config.JID = make(map[string]string)
	}
	config.configFilePath = configFilePath
	return config
}

func (sc *IServerConfig) GetConfigPath() string {
	return sc.configFilePath
}
func (sc *IServerConfig) SetConfigPath(path string) *IServerConfig {
	sc.configFilePath = path
	return sc
}
func (sc *IServerConfig) Save() {
	byteJson, err := json.MarshalIndent(sc, "", "    ")
	if err != nil {
		return
	}
	err = os.WriteFile(sc.configFilePath, byteJson, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
