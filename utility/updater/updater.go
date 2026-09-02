package updater

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

type KVResponse struct {
	Version int    `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

func GetFileKey(envName, component, osName, arch string) string {
	return fmt.Sprintf("%s_%s_%s_%s", envName, component, osName, arch)
}

func CheckAndUpdate(envName, kvBaseURL, componentName string, currentVersion int) (bool, error) {
	kvKey := GetFileKey(envName, componentName, runtime.GOOS, runtime.GOARCH)
	kvServerURL := kvBaseURL + kvKey

	log.Printf("Checking for updates: %s", kvKey)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", kvServerURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("KV server unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("KV server returned: %d", resp.StatusCode)
	}

	var kvData KVResponse
	if err := json.NewDecoder(resp.Body).Decode(&kvData); err != nil {
		return false, err
	}

	if currentVersion >= kvData.Version {
		return false, nil // Up to date
	}

	log.Printf("Update found! v%d -> v%d", currentVersion, kvData.Version)

	// os.Executable() perfectly handles user-renamed binaries
	exePath, err := os.Executable()
	if err != nil {
		return false, err
	}

	gzTmpPath := exePath + ".gz.tmp"
	binTmpPath := exePath + ".tmp"
	defer os.Remove(gzTmpPath)
	defer os.Remove(binTmpPath)

	// Download
	downReq, _ := http.NewRequest("GET", kvData.URL, nil)
	downReq.Header.Set("Accept-Encoding", "identity") // Prevent auto-decompression
	downloadResp, err := client.Do(downReq)
	if err != nil || downloadResp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("download failed: %w", err)
	}
	defer downloadResp.Body.Close()

	gzFile, err := os.Create(gzTmpPath)
	if err != nil {
		return false, err
	}

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(gzFile, h), downloadResp.Body); err != nil {
		gzFile.Close()
		return false, err
	}
	gzFile.Close()

	if hex.EncodeToString(h.Sum(nil)) != kvData.SHA256 {
		return false, fmt.Errorf("hash mismatch")
	}

	// Extract
	if err := extractGzip(gzTmpPath, binTmpPath); err != nil {
		return false, err
	}

	if runtime.GOOS != "windows" {
		os.Chmod(binTmpPath, 0755)
	} else {
		os.Rename(exePath, exePath+".old")
		defer os.Remove(exePath + ".old")
	}

	if err := os.Rename(binTmpPath, exePath); err != nil {
		return false, err
	}

	return true, nil
}

func extractGzip(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	gr, err := gzip.NewReader(sf)
	if err != nil {
		return err
	}
	defer gr.Close()

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer df.Close()

	_, err = io.Copy(df, gr)
	return err
}
