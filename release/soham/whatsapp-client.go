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

	if len(os.Args) < 2 {
		fmt.Println("Usage: builder <version>")
		return
	}
	version := os.Args[1]
	err := os.MkdirAll("build", 0755)
	if err != nil {
		panic(err)
	}
	buildFilePath := "./servers/soham/whatsapp-client/main.go"

	if slices.Contains(os.Args, "--dev") {
		buildFilePath = "../../servers/soham/whatsapp-client/main.go"
	}

	serverBinary := "whatsapp-client.o"
	if runtime.GOOS == "windows" {
		archs = append(archs, "386")
		serverBinary = "whatsapp-client.exe"
	}
	serverBinaryPath := filepath.Join("build", serverBinary)
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

		fmt.Println("Calculating SHA256...")

		hash, err := sha256File(serverBinaryPath)
		if err != nil {
			panic(err)
		}
		// https://files.rpso.in/static/soham/wbot/darwin_amd64/whatsapp-client.o
		versionInfo := VersionInfo{
			Version: atoi(version),
			URL:     fmt.Sprintf("https://files.rpso.in/static/soham/wbot/%s_%s/%s", runtime.GOOS, arch, serverBinary),
			SHA256:  hash,
		}

		data, _ := json.MarshalIndent(versionInfo, "", " ")

		fmt.Println("Uploading server binary...")
		err = uploadFile(serverBinaryPath, serverBinary, fmt.Sprintf("soham/wbot/%s_%s", runtime.GOOS, arch))
		// uploadFile(serverBinaryPath, "https://fileserver.com/server_v"+version)
		if err != nil {
			panic(err)
		}

		fmt.Println("Updating keyvalue store...")

		err = updateKeyValue(fmt.Sprintf("soham_go_wbot_%s_%s", runtime.GOOS, arch), data)
		if err != nil {
			panic(err)
		}
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
		Timeout: time.Second * 120,
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
