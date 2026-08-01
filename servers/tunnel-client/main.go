package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	chclient "github.com/jpillora/chisel/client"
	chshare "github.com/jpillora/chisel/share"
	"github.com/jpillora/chisel/share/cos"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/validator"
)

type (
	tunnelService struct{}
	Config        struct {
		ServerIp           string         `json:"serverIp" validate:"required"`
		ServerPort         int            `json:"serverPort" validate:"required,port"`
		RemoteTunnelConfig []TunnelConfig `json:"remoteTunnelConfig" validate:"required,min=1"`
	}
	TunnelConfig struct {
		LocalPort  int `json:"localPort" validate:"required,port"`
		RemotePort int `json:"remotePort" validate:"required,port"`
	}
)

var (
	Client          *chclient.Client
	ChiselContext   context.Context
	ShouldBeRunning = false
)

func main() {
	f, err := os.OpenFile(filepath.Join(FindAndReturnCurrentDir(), "debug.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln(fmt.Errorf("error opening file: %v", err))
	}
	defer f.Close()

	log.SetOutput(f)
	start()
	log.Print("Finished Function")
}

func start() {
	fmt.Println(len(os.Args), os.Args)
	currentDir := FindAndReturnCurrentDir()
	configFilePAth := filepath.Join(currentDir, "configs.json")
	ThisConfig := &Config{}

	fmt.Printf("Config path %s\n", configFilePAth)
	if exist, _ := utility_functions.Exist(configFilePAth); !exist {
		panic(fmt.Errorf("Config Not Exist on Path %s", configFilePAth))
	}
	dat, err := os.ReadFile(configFilePAth)
	Check(err)
	json.Unmarshal(dat, ThisConfig)

	if errs := validator.Validator.Validate(ThisConfig); len(errs) > 0 {
		panic(fmt.Errorf("Config Error %#v", errs))
	}

	remote := make([]string, len(ThisConfig.RemoteTunnelConfig))
	for i, r := range ThisConfig.RemoteTunnelConfig {
		remote[i] = fmt.Sprintf("R:%d:%d", r.RemotePort, r.LocalPort)
	}
	chshare.BuildVersion = "1.9.1"
	client, err := chclient.NewClient(&chclient.Config{
		Remotes:          remote,
		MaxRetryCount:    100,
		MaxRetryInterval: 60 * time.Minute,
		Server:           fmt.Sprintf("%s:%d", ThisConfig.ServerIp, ThisConfig.ServerPort),
	})
	if err != nil {
		log.Print(err.Error())
	}
	Client = client
	ChiselContext = cos.InterruptContext()
	StartChisel()
}

func StartChisel() {
	ShouldBeRunning = true
	runningChisel()
}

func runningChisel() {
	time.Sleep(3 * time.Second)
	if ShouldBeRunning {
		log.Print("Chisel Started...!")
		Client.Start(ChiselContext)
		Client.Wait()
		log.Print("Chisel Stopped...!")
		runningChisel()
	}
}

func StopChisel() {
	ShouldBeRunning = false
	Client.Close()
	log.Print("Chisel Stopped...!")
}

func IsDebug() bool {
	return slices.Contains(os.Args, "--dev")
}

func FindAndReturnCurrentDir() string {
	currentDir := ""
	if IsDebug() {
		current, err := os.Getwd()
		Check(err)
		currentDir = current
	} else {
		exePath, err := os.Executable()
		currentDir = filepath.Dir(exePath)
		Check(err)
	}
	return currentDir
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}
