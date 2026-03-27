package soham_whatsapp_gui_config

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

const QRCODEURL = "http://localhost:4000/v1/qr_code"
const LoginStatusURL = "http://localhost:4000/v1/status"

type LoginStatusApiReponse struct {
	Status int `json:"status"`
}

type QrCodeApiCallResponse struct {
	QrCode     string `json:"qrCode"`
	QrCodeData string `json:"qrCodeData"`
}

func init() {
	CurrentDirectory := env.FindAndReturnCurrentDir()
	ServerConfigFilePath = filepath.Join(CurrentDirectory, whatsapp_config.ServerConfigFileName)
}

func ValidateConfig() (bool, *whatsapp_interfaces.IServerConfig) {
	if _, err := utility_functions.Exist(ServerConfigFilePath); errors.Is(err, os.ErrNotExist) {
		// panic(fmt.Errorf("CONFIG_NOT_EXIST_ON_PATH %s", ServerConfigFilePath))
		return false, nil
	}
	config, err := readConfigFileAndReturniserverConfig(ServerConfigFilePath)
	if err != nil {
		return false, nil
		// panic(err)
	}
	return true, config
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

func QrCodeApiCall(token string) (bool, string) {
	req, err := http.NewRequest("GET", QRCODEURL, nil)

	if err != nil {
		fmt.Println(err)
		return false, ""
	}
	req.Header.Add("X-Api-Token", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return false, ""
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return false, ""
	}
	if strings.Contains(string(body), "success") {
		return false, ""
	}
	qrcodeRespo := new(QrCodeApiCallResponse)
	err = json.Unmarshal(body, qrcodeRespo)
	if err != nil {
		fmt.Println(err)
		return false, ""
	}
	return true, qrcodeRespo.QrCode
}
func LoginApiCall(token string) (bool, error) {
	req, err := http.NewRequest("GET", LoginStatusURL, nil)

	if err != nil {
		fmt.Println(err)
		return false, err
	}
	req.Header.Add("X-Api-Token", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	statusRespo := new(LoginStatusApiReponse)
	err = json.Unmarshal(body, statusRespo)
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	if statusRespo.Status == 1 {
		return true, nil
	} else {
		return false, nil

	}
}
