package functions

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
	"time"

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

func Sha256File(path string) (string, error) {
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

func getExePath() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current working directory if execution lookup fails
		cwd, _ := os.Getwd()
		return cwd
	}
	// Evaluates symlinks cleanly to guarantee correct execution paths
	realPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		return realPath
	}
	return exePath
}

func getExeDir() string {
	return filepath.Dir(getExePath())
}
func CheckAndDownload(versinoEndPoint func() string) string {
	// 1. Get the absolute path of the running executable
	// exePath, err := os.Executable()
	// if err != nil {
	// 	fmt.Printf("Error getting executable path: %v\n", err)
	// 	return
	// }

	// // 2. Extract just the binary name from the full path
	// binName := filepath.Base(exePath)

	// 1. Resolve target file paths relative to where the current application is running
	appDir := getExeDir()
	versionFilePath := filepath.Join(appDir, VersionFileName)

	serverBinaryName := filepath.Base(getExePath())

	// Anchor the target binary to the execution directory
	serverBinary := filepath.Join(appDir, serverBinaryName)

	if checkAndRunCalled {
		return serverBinary
	}

	endpoint := versinoEndPoint()
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

		hash, err := Sha256File(gzipFile)
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
	// backup existing binary
	backup := serverBinary + ".old"
	os.Remove(backup)
	if exist, _ := utility_functions.Exist(serverBinary); exist {
		os.Rename(serverBinary, backup)
	}
	// move new binary
	err := utility_functions_gzip.GzipDecompressFile(tmpFile, serverBinary)
	if err != nil {
		return err
	}

	// macOS / Linux need execute permission
	if runtime.GOOS != "windows" {
		os.Chmod(serverBinary, 0755)
	}

	// cleanup backup
	os.Remove(backup)

	return nil
}
