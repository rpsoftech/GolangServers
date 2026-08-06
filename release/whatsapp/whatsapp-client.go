package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	utility_functions_gzip "github.com/rpsoftech/golang-servers/utility/functions/gzip"
)

const (
	fileServerURL = "https://files.rpso.in/upload/"
	keyValueURL   = "https://keyvalue.rpso.in/public/"
)

var (
	fileServerToken = os.Getenv("FILE_SERVER_TOKEN")
	kvToken         = os.Getenv("KV_TOKEN")
	archs           = []string{"amd64", "arm64"}
)

type VersionInfo struct {
	Version int    `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

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

		hash, err := sha256File(gzipFilePath)
		if err != nil {
			panic(err)
		}
		// https://files.rpso.in/static/soham/wbot/darwin_amd64/whatsapp-client.o
		versionInfo := VersionInfo{
			Version: atoi(version),
			URL:     fmt.Sprintf("https://files.rpso.in/static/whatsapp/%s_%s/%s", runtime.GOOS, arch, gzipFileName),
			SHA256:  hash,
		}

		data, _ := json.MarshalIndent(versionInfo, "", " 	")

		fmt.Println("Uploading server binary...")
		err = uploadFile(gzipFilePath, gzipFileName, fmt.Sprintf("whatsapp/%s_%s", runtime.GOOS, arch))
		// uploadFile(serverBinaryPath, "https://fileserver.com/server_v"+version)
		if err != nil {
			panic(err)
		}

		fmt.Println("Updating keyvalue store...")

		err = updateKeyValue(fmt.Sprintf("whatsapp_go_wbot_%s_%s", runtime.GOOS, arch), data)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Uploaded %s", data)
		// uploadFile(versionFile, "https://kvserver.com/version.json")
	}
	fmt.Println("Build and upload complete")
}

func sha256File(path string) (string, error) {

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func uploadFile(path string, filename string, uploadPath string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	part, err := writer.CreateFormFile(filename, filepath.Base(path))
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: time.Second * 540,
	}
	io.Copy(part, file)

	writer.WriteField("path", uploadPath)

	writer.Close()

	req, err := http.NewRequest(
		"POST",
		fileServerURL+filename,
		payload,
	)

	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+fileServerToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(body))
	fmt.Println("Uploaded:", filename)

	return nil
}

func updateKeyValue(key string, data []byte) error {
	req, err := http.NewRequest(
		"POST",
		keyValueURL+key,
		bytes.NewBuffer(data),
	)

	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+kvToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	fmt.Println("KeyValue updated:", key)

	return nil
}

func atoi(s string) int {

	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
