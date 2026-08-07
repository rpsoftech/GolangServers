package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/rpsoftech/golang-servers/functions"
	"github.com/rpsoftech/golang-servers/release"
	utility_functions_gzip "github.com/rpsoftech/golang-servers/utility/functions/gzip"
)

var (
	fileServerToken = os.Getenv("FILE_SERVER_TOKEN")
	kvToken         = os.Getenv("KV_TOKEN")
	archs           = []string{"amd64", "arm64"}
)

func main() {
	fmt.Printf("File token length: %d\n", len(fileServerToken))
	fmt.Printf("Keyvalue token length: %d\n", len(kvToken))
	version := ""
	if len(os.Args) < 2 {
		if os.Getenv("VERSION") == "" {
			fmt.Println("Usage: builder <version>")
			return
		}
		version = os.Getenv("VERSION")
	} else {
		version = os.Args[1]
	}
	fmt.Printf("Version: %s\n", version)
	err := os.MkdirAll("build", 0755)
	if err != nil {
		panic(err)
	}
	buildFilePath := "./servers/whatsapp-server/main.go"

	if slices.Contains(os.Args, "--dev") {
		buildFilePath = "../../servers/whatsapp-server/main.go"
	}

	serverBinaryName := "whatsapp-client.o"
	if runtime.GOOS == "windows" {
		archs = append(archs, "386")
		serverBinaryName = "whatsapp-client.exe"
	}
	serverBinaryPath := filepath.Join("build", serverBinaryName)
	for _, arch := range archs {
		fmt.Printf("Building server for %s...", arch)
		cmd := exec.Command(
			"go",
			"build",
			"-ldflags",
			fmt.Sprintf("-s -w -X main.version=%s", version),
			"-o",
			serverBinaryPath,
			buildFilePath,
		)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOARCH=%s", arch))
		log.Printf("running command %s", cmd.String())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		if err != nil {
			panic(err)
		}
		gzipFilePath := serverBinaryPath + ".gz"
		gzipFileName := serverBinaryName + ".gz"
		utility_functions_gzip.GzipCompressFile(serverBinaryPath, gzipFilePath)
		fmt.Println("Calculating SHA256...")

		hash, err := functions.Sha256File(gzipFilePath)
		if err != nil {
			panic(err)
		}
		versionInfo := release.VersionInfo{
			Version: release.Atoi(version),
			URL:     fmt.Sprintf("https://files.rpso.in/static/whatsapp_client/%s_%s/%s", runtime.GOOS, arch, gzipFileName),
			SHA256:  hash,
		}

		data, _ := json.MarshalIndent(versionInfo, "", " 	")

		fmt.Println("Uploading server binary...")
		err = release.UploadFile(gzipFilePath, gzipFileName, fmt.Sprintf("whatsapp_client/%s_%s", runtime.GOOS, arch), release.FileServerURL, fileServerToken)
		// uploadFile(serverBinaryPath, "https://fileserver.com/server_v"+version)
		if err != nil {
			panic(err)
		}

		fmt.Println("Updating keyvalue store...")

		err = release.UpdateKeyValue(fmt.Sprintf("whatsapp_go_wbot_%s_%s", runtime.GOOS, arch), data, release.KeyValueURL, kvToken)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Uploaded %s", data)
		// uploadFile(versionFile, "https://kvserver.com/version.json")
	}
	fmt.Println("Build and upload complete")
}
