package whatsapp_config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rpsoftech/golang-servers/env"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

var (
	Env                     *whatsapp_interfaces.EnvInterface
	WhatsappNumberConfigMap *whatsapp_interfaces.IServerConfig
	WhatsappNumberToIDMap   map[string]string
	CurrentDirectory        string = ""
	serverConfigFilePath    string = ""
)

const ServerConfigFileName = "server.config.json"

func InitaliseWhatsappEnvAndConfig() {
	CurrentDirectory = env.FindAndReturnCurrentDir()
	serverConfigFilePath = filepath.Join(CurrentDirectory, ServerConfigFileName)
	if _, err := utility_functions.Exist(serverConfigFilePath); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("CONFIG_NOT_EXIST_ON_PATH %s", serverConfigFilePath))
	}
	WhatsappNumberConfigMap = whatsapp_interfaces.ReadConfigFileAndReturniserverConfig(serverConfigFilePath)
	allow_local_no_Auth, err := strconv.ParseBool(os.Getenv(whatsapp_interfaces.Allow_local_no_auth_KEY))
	if err != nil {
		log.Fatal(err)
	}
	auto_connect_to_whatsapp, err := strconv.ParseBool(os.Getenv(whatsapp_interfaces.Auto_Connect_To_Whatsapp_KEY))
	if err != nil {
		log.Fatal(err)
	}
	open_browser_for_scan_KEY, err := strconv.ParseBool(os.Getenv(whatsapp_interfaces.Open_browser_for_scan_KEY))
	if err != nil {
		log.Fatal(err)
	}

	Env = &whatsapp_interfaces.EnvInterface{
		ALLOW_LOCAL_NO_AUTH:      allow_local_no_Auth,
		AUTO_CONNECT_TO_WHATSAPP: auto_connect_to_whatsapp,
		OPEN_BROWSER_FOR_SCAN:    open_browser_for_scan_KEY,
	}
	WhatsappNumberToIDMap = map[string]string{}
	env.ValidateEnv(Env)
}
