package whatsapp_config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/validator"
)

var (
	Env                     *EnvInterface
	WhatsappNumberConfigMap *IServerConfig
	CurrentDirectory        string = ""
	serverConfigFilePath    string = ""
)

const ServerConfigFileName = "server.config.json"

type (
	EnvInterface struct {
		ALLOW_LOCAL_NO_AUTH      bool `json:"ALLOW_LOCAL_NO_AUTH" validate:"boolean"`
		AUTO_CONNECT_TO_WHATSAPP bool `json:"AUTO_CONNECT_TO_WHATSAPP" validate:"boolean"`
		OPEN_BROWSER_FOR_SCAN    bool `json:"OPEN_BROWSER_FOR_SCAN" validate:"boolean"`
	}
	IServerConfig struct {
		Tokens map[string]string `json:"tokens" validate:"required"`
		JID    map[string]string `json:"JID"`
	}
)

func init() {
	CurrentDirectory = env.FindAndReturnCurrentDir()
	WhatsappNumberConfigMap = ReadConfigFileAndReturnIt(CurrentDirectory)
	allow_local_no_Auth, err := strconv.ParseBool(os.Getenv(allow_local_no_auth_KEY))
	if err != nil {
		log.Fatal(err)
	}
	auto_connect_to_whatsapp, err := strconv.ParseBool(os.Getenv(Auto_Connect_To_Whatsapp_KEY))
	if err != nil {
		log.Fatal(err)
	}
	open_browser_for_scan_KEY, err := strconv.ParseBool(os.Getenv(open_browser_for_scan_KEY))
	if err != nil {
		log.Fatal(err)
	}

	Env = &EnvInterface{
		ALLOW_LOCAL_NO_AUTH:      allow_local_no_Auth,
		AUTO_CONNECT_TO_WHATSAPP: auto_connect_to_whatsapp,
		OPEN_BROWSER_FOR_SCAN:    open_browser_for_scan_KEY,
	}
	errs := validator.Validator.Validate(Env)
	if len(errs) > 0 {
		println(errs)
		panic(errs[0])
	}
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
	if _, err := os.Stat(configFilePAth); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("CONFIG_NOT_EXIST_ON_PATH %s", configFilePAth))
	}
	dat, err := os.ReadFile(configFilePAth)
	env.Check(err)
	err = json.Unmarshal(dat, config)
	env.Check(err)
	if errs := validator.Validator.Validate(config); len(errs) > 0 {
		panic(fmt.Errorf("CONFIG_ERROR %#v", errs))
	}
	if config.JID == nil {
		config.JID = make(map[string]string)
	}
	return config
}
