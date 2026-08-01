package sohan_whatsapp_auto_download

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	sohan_whatsapp_keys "github.com/rpsoftech/golang-servers/apps/sohan/whatsapp/keys"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	utility_functions_gzip "github.com/rpsoftech/golang-servers/utility/functions/gzip"
)

const VersionFileName = "client-version.json"

type VersionInfo struct {
	Version int    `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

type progressReader struct {
	reader io.Reader
	total  int64
	read   int64
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	p.read += int64(n)

	if p.total > 0 {
		percentage := float64(p.read) / float64(p.total) * 100
		fmt.Printf("\rDownloading update: %.2f%% (%d/%d bytes)", percentage, p.read, p.total)
	} else {
		fmt.Printf("\rDownloading update: %d bytes received", p.read)
	}
	return n, err
}

var checkAndRunCalled = false

// getExeDir returns the absolute path of the directory where this executable is running
func getExeDir() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current working directory if execution lookup fails
		cwd, _ := os.Getwd()
		return cwd
	}
	// Evaluates symlinks cleanly to guarantee correct execution paths
	realPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		return filepath.Dir(realPath)
	}
	return filepath.Dir(exePath)
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

func GetVersionEndpoint() string {
	switch fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH) {
	case "windows_amd64":
		return "https://keyvalue.rpso.in/public/soham_go_wbot_windows_amd64"
	case "darwin_amd64":
		return "https://keyvalue.rpso.in/public/soham_go_wbot_darwin_amd64"
	case "darwin_arm64":
		return "https://keyvalue.rpso.in/public/soham_go_wbot_darwin_arm64"
	default:
		return ""
	}
}

func CheckAndDownload() string {
	// 1. Resolve target file paths relative to where the current application is running
	appDir := getExeDir()
	versionFilePath := filepath.Join(appDir, VersionFileName)

	serverBinaryName := "whatsapp-client.o"
	if runtime.GOOS == "windows" {
		serverBinaryName = "whatsapp-client.exe"
	}

	// Anchor the target binary to the execution directory
	serverBinary := filepath.Join(appDir, serverBinaryName)

	if checkAndRunCalled {
		return serverBinary
	}

	endpoint := GetVersionEndpoint()
	if endpoint == "" {
		log.Printf("Fatal: Endpoint Not Found For OS: %s, Arch: %s\n", runtime.GOOS, runtime.GOARCH)
		return serverBinary
	}

	resp, err := http.Get(endpoint)
	if err != nil {
		log.Println("Error: Failed to poll client metadata version endpoint.")
		return serverBinary
	}
	defer resp.Body.Close()

	var cloud VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&cloud); err != nil {
		log.Println("Error parsing remote version payload:", err)
		return serverBinary
	}

	var local VersionInfo
	data, err := os.ReadFile(versionFilePath)
	if err == nil {
		json.Unmarshal(data, &local)
	}

	gzipFile := serverBinary + ".gz"
	needDownload := false

	if exist, _ := utility_functions.Exist(serverBinary); !exist {
		needDownload = true
	}
	if local.Version != cloud.Version {
		needDownload = true
	}

	if needDownload {
		os.Remove(gzipFile)
		log.Printf("New version found (Local: %d, Cloud: %d). Initializing download...", local.Version, cloud.Version)

		err := downloadFileWithProgress(cloud.URL, gzipFile)
		fmt.Println()
		if err != nil {
			log.Println("Error: File download failed:", err)
			return serverBinary
		}

		hash, err := sha256File(gzipFile)
		if err != nil {
			log.Println("Error hashing temporary file:", err)
			return serverBinary
		}

		if hash != cloud.SHA256 {
			log.Printf("Error: Checksum mismatch. Expected %s, got %s\n", cloud.SHA256, hash)
			os.Remove(gzipFile)
			return serverBinary
		}

		err = replaceBinarySafe(gzipFile, serverBinary)
		if err != nil {
			log.Println("Binary unpack and replacement transaction failed:", err)
			return serverBinary
		}

		vdata, _ := json.Marshal(cloud)
		os.WriteFile(versionFilePath, vdata, 0644)
		log.Println("Update deployed successfully to base execution folder.")
		os.Exit(0)
	}

	checkAndRunCalled = true
	return serverBinary
}

func downloadFileWithProgress(url string, targetPath string) error {
	client := &http.Client{
		Timeout: 500 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	pr := &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
	}

	_, err = io.Copy(out, pr)
	return err
}

func replaceBinarySafe(tmpFile string, serverBinary string) error {
	if sohan_whatsapp_keys.ServerCmd != nil && sohan_whatsapp_keys.ServerCmd.Process != nil {
		sohan_whatsapp_keys.ServerCmd.Process.Kill()
		time.Sleep(2 * time.Second)
	}

	backup := serverBinary + ".old"
	os.Remove(backup)

	if exist, _ := utility_functions.Exist(serverBinary); exist {
		err := os.Rename(serverBinary, backup)
		if err != nil {
			if !strings.Contains(err.Error(), "access is denied") {
				return fmt.Errorf("failed to cycle target out of space: %w", err)
			}
		}
	}

	err := utility_functions_gzip.GzipDecompressFile(tmpFile, serverBinary)
	if err != nil {
		if exist, _ := utility_functions.Exist(backup); exist {
			os.Rename(backup, serverBinary)
		}
		return fmt.Errorf("decompression step failure: %w", err)
	}

	if runtime.GOOS != "windows" {
		os.Chmod(serverBinary, 0755)
	}

	os.Remove(backup)
	os.Remove(tmpFile)

	return nil
}
