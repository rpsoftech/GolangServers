package soham_whatsapp_gui_config

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/google/uuid"
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
		return nil, fmt.Errorf("CONFIG_ERROR %#v", errs)
	}
	if len(config.Tokens) == 0 {
		return nil, errors.New("NO_TOKENS")
	}
	for token := range config.Tokens {
		if _, err := ValidUUID(token); err != "" {
			return nil, errors.New(err)
		}
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

func CreateQRFromBase64(base64Str string) (*canvas.Image, error) {

	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	qr := canvas.NewImageFromImage(img)

	qr.FillMode = canvas.ImageFillContain

	qr.SetMinSize(
		fyne.NewSize(200, 200),
	)

	return qr, nil
}

//////////////////////////////////////////////////////
// UUID VALIDATION
//////////////////////////////////////////////////////

func ValidUUID(uuidstring string) (bool, string) {
	u, err := uuid.Parse(uuidstring)
	if err != nil {
		return false, fmt.Sprintf("String %q is invalid: %v\n", uuidstring, err)
	}
	// Check if it is specifically version 5
	if u.Version() == 5 {
		return true, ""
	} else {
		return false, fmt.Sprintf("It is UUID Version %d.\n", u.Version())
	}
}
