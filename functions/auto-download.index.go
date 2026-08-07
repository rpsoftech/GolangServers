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
	log.Printf("[Updater] Calculating SHA256 checksum for file: %s", path)
	file, err := os.Open(path)
	if err != nil {
		log.Printf("[Updater] Error opening file for hashing (%s): %v", path, err)
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		log.Printf("[Updater] Error copying file stream to hash writer (%s): %v", path, err)
		return "", err
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	log.Printf("[Updater] Computed SHA256 (%s): %s", path, checksum)
	return checksum, nil
}

func getExePath() string {
	exePath, err := os.Executable()
	if err != nil {
		cwd, _ := os.Getwd()
		log.Printf("[Updater] Warning: Failed to look up executable path, falling back to CWD (%s): %v", cwd, err)
		return cwd
	}

	realPath, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		return realPath
	}

	log.Printf("[Updater] Warning: Failed to evaluate symlinks for executable (%s): %v", exePath, err)
	return exePath
}

func getExeDir() string {
	dir := filepath.Dir(getExePath())
	log.Printf("[Updater] Resolved execution directory: %s", dir)
	return dir
}

func CheckAndDownload(versinoEndPoint func() string) string {
	log.Printf("[Updater] Initializing update check on OS: %s, Arch: %s", runtime.GOOS, runtime.GOARCH)

	appDir := getExeDir()
	versionFilePath := filepath.Join(appDir, VersionFileName)
	serverBinaryName := filepath.Base(getExePath())
	serverBinary := filepath.Join(appDir, serverBinaryName)

	log.Printf("[Updater] Binary target: %s", serverBinary)
	log.Printf("[Updater] Local version file path: %s", versionFilePath)

	if checkAndRunCalled {
		log.Println("[Updater] Update check already executed in this process lifetime. Skipping.")
		return serverBinary
	}

	endpoint := versinoEndPoint()
	if endpoint == "" {
		log.Printf("[Updater] Fatal: Update endpoint URL is empty for OS: %s, Arch: %s\n", runtime.GOOS, runtime.GOARCH)
		return serverBinary
	}
	log.Printf("[Updater] Fetching update metadata from endpoint: %s", endpoint)

	resp, err := http.Get(endpoint)
	if err != nil {
		log.Printf("[Updater] Error: Failed to poll update metadata endpoint (%s): %v", endpoint, err)
		return serverBinary
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Updater] Error: Endpoint returned non-200 status code: %d %s", resp.StatusCode, resp.Status)
		return serverBinary
	}

	var cloud VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&cloud); err != nil {
		log.Printf("[Updater] Error parsing remote version JSON response: %v", err)
		return serverBinary
	}
	log.Printf("[Updater] Remote metadata received -> Version: %d, URL: %s, SHA256: %s", cloud.Version, cloud.URL, cloud.SHA256)

	var local VersionInfo
	data, err := os.ReadFile(versionFilePath)
	if err == nil {
		if err := json.Unmarshal(data, &local); err != nil {
			log.Printf("[Updater] Warning: Failed to parse local version file (%s): %v", versionFilePath, err)
		} else {
			log.Printf("[Updater] Local version metadata read successfully -> Version: %d", local.Version)
		}
	} else {
		log.Printf("[Updater] Local version file not found or unreadable (%s). Assuming version 0. Error: %v", versionFilePath, err)
	}

	gzipFile := serverBinary + ".gz"
	needDownload := false

	binaryExists, _ := utility_functions.Exist(serverBinary)
	if !binaryExists {
		log.Printf("[Updater] Target binary does not exist at %s. Triggering download.", serverBinary)
		needDownload = true
	}

	if local.Version != cloud.Version {
		log.Printf("[Updater] Version mismatch detected (Local: %d, Remote: %d). Triggering download.", local.Version, cloud.Version)
		needDownload = true
	}

	if needDownload {
		log.Printf("[Updater] Removing stale temporary download file if present: %s", gzipFile)
		os.Remove(gzipFile)

		log.Printf("[Updater] New version required (Local: %d, Cloud: %d). Initializing download sequence...", local.Version, cloud.Version)

		err := downloadFileWithProgress(cloud.URL, gzipFile)
		fmt.Println() // Print newline to clear terminal progress bar line
		if err != nil {
			log.Printf("[Updater] Error: Update package download failed: %v", err)
			return serverBinary
		}

		hash, err := Sha256File(gzipFile)
		if err != nil {
			log.Printf("[Updater] Error hashing downloaded update package (%s): %v", gzipFile, err)
			return serverBinary
		}

		if hash != cloud.SHA256 {
			log.Printf("[Updater] Error: Checksum mismatch for downloaded package! Expected %s, calculated %s", cloud.SHA256, hash)
			log.Printf("[Updater] Cleaning up corrupt download archive: %s", gzipFile)
			os.Remove(gzipFile)
			return serverBinary
		}
		log.Println("[Updater] Package SHA256 verification passed.")

		err = replaceBinarySafe(gzipFile, serverBinary)
		if err != nil {
			log.Printf("[Updater] Binary unpack and replacement transaction failed: %v", err)
			return serverBinary
		}

		vdata, err := json.Marshal(cloud)
		if err != nil {
			log.Printf("[Updater] Warning: Failed to marshal new version info to JSON: %v", err)
		} else {
			if err := os.WriteFile(versionFilePath, vdata, 0644); err != nil {
				log.Printf("[Updater] Warning: Failed to write updated version file (%s): %v", versionFilePath, err)
			} else {
				log.Printf("[Updater] Successfully saved local version metadata file: %s", versionFilePath)
			}
		}

		log.Println("[Updater] Update deployed successfully to base execution folder. Exiting process for restart...")
		os.Exit(0)
	}

	log.Println("[Updater] Application is up to date. No update required.")
	checkAndRunCalled = true
	return serverBinary
}

func downloadFileWithProgress(url string, targetPath string) error {
	log.Printf("[Updater] Initiating HTTP GET request to download package: %s", url)
	client := &http.Client{
		Timeout: 500 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[Updater] Download request failed for URL (%s): %v", url, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status code: %d %s", resp.StatusCode, resp.Status)
		log.Printf("[Updater] Download failed for URL (%s): %v", url, err)
		return err
	}

	log.Printf("[Updater] Creating local temporary destination file: %s", targetPath)
	out, err := os.Create(targetPath)
	if err != nil {
		log.Printf("[Updater] Failed to create local target file (%s): %v", targetPath, err)
		return err
	}
	defer out.Close()

	log.Printf("[Updater] Downloading payload (Content Length: %d bytes)...", resp.ContentLength)
	pr := &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
	}

	written, err := io.Copy(out, pr)
	if err != nil {
		log.Printf("\n[Updater] Error streaming download contents to file (%s): %v", targetPath, err)
		return err
	}

	log.Printf("\n[Updater] Download completed successfully. Total bytes written: %d", written)
	return nil
}

func replaceBinarySafe(tmpFile string, serverBinary string) error {
	backup := serverBinary + ".old"
	log.Printf("[Updater] Starting safe binary replacement process.")
	log.Printf("[Updater] Cleaning up old backup file if it exists: %s", backup)
	os.Remove(backup)

	if exist, _ := utility_functions.Exist(serverBinary); exist {
		log.Printf("[Updater] Backing up active binary (%s -> %s)", serverBinary, backup)
		if err := os.Rename(serverBinary, backup); err != nil {
			log.Printf("[Updater] Failed to rename active binary to backup (%s -> %s): %v", serverBinary, backup, err)
			return err
		}
	} else {
		log.Printf("[Updater] No existing binary found at %s to backup.", serverBinary)
	}

	log.Printf("[Updater] Decompressing update package (%s -> %s)", tmpFile, serverBinary)
	err := utility_functions_gzip.GzipDecompressFile(tmpFile, serverBinary)
	if err != nil {
		log.Printf("[Updater] Error decompressing gzip archive (%s): %v", tmpFile, err)
		if exist, _ := utility_functions.Exist(backup); exist {
			log.Printf("[Updater] Attempting rollback: Restoring backup (%s -> %s)", backup, serverBinary)
			os.Rename(backup, serverBinary)
		}
		return err
	}

	if runtime.GOOS != "windows" {
		log.Printf("[Updater] Setting POSIX executable permissions (0755) on %s", serverBinary)
		if err := os.Chmod(serverBinary, 0755); err != nil {
			log.Printf("[Updater] Warning: Failed to set chmod 0755 permissions on %s: %v", serverBinary, err)
		}
	}

	log.Printf("[Updater] Cleaning up backup file: %s", backup)
	os.Remove(backup)

	log.Printf("[Updater] Removing temporary update package: %s", tmpFile)
	os.Remove(tmpFile)

	log.Println("[Updater] Safe binary replacement completed successfully.")
	return nil
}
