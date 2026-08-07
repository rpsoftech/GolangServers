package soham_whatsapp_auto_download

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	soham_whatsapp_keys "github.com/rpsoftech/golang-servers/apps/sohan/whatsapp/keys"
	"github.com/rpsoftech/golang-servers/functions"
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
	reader   io.Reader
	total    int64
	read     int64
	progress *widget.ProgressBar
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	p.read += int64(n)
	if p.total > 0 {
		value := float64(p.read) / float64(p.total)
		fyne.Do(func() {
			p.progress.SetValue(value)
		})
	}
	return n, err
}

var checkAndRunCalled = false

func GetVersionEndpoint() string {

	switch fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH) {
	// soham_go_wbot_darwin_amd64
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

func CheckAndDownload(progress *widget.ProgressBar, win fyne.Window) string {
	serverBinary := "whatsapp-client.o"
	if runtime.GOOS == "windows" {
		serverBinary = "whatsapp-client.exe"
	}
	serverBinary = filepath.Join(soham_whatsapp_keys.ConfigDir, serverBinary)
	if checkAndRunCalled {
		return serverBinary
	}
	endpoint := GetVersionEndpoint()
	if endpoint == "" {
		panic(fmt.Errorf("Endpoint Not Found For %s and %s", runtime.GOOS, runtime.GOARCH))
	}
	resp, err := http.Get(endpoint)
	if err != nil {
		fmt.Println("Error in Downloading client file")
		panic(err)
	}
	defer resp.Body.Close()
	var cloud VersionInfo
	json.NewDecoder(resp.Body).Decode(&cloud)

	var local VersionInfo

	data, err := os.ReadFile(VersionFileName)
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
		fyne.DoAndWait(func() {
			win.Show()
		})
		err := downloadFileWithProgress(cloud.URL, gzipFile, progress)
		if err != nil {
			if exist, _ := utility_functions.Exist(serverBinary); !exist {
				panic(fmt.Errorf("File Downloading Failed"))
				// needDownload = true
			}
			return serverBinary
		}

		hash, err := functions.Sha256File(gzipFile)
		if err != nil {
			return serverBinary
		}

		if hash != cloud.SHA256 {
			os.Remove(gzipFile)
			return serverBinary
		}
		err = replaceBinarySafe(gzipFile, serverBinary)
		if err != nil {
			log.Println("Binary replace failed:", err)
			return serverBinary
		}
		vdata, _ := json.Marshal(cloud)
		os.WriteFile(VersionFileName, vdata, 0644)
	}
	checkAndRunCalled = true
	return serverBinary
}

func downloadFileWithProgress(url string, filepath string, progress *widget.ProgressBar) error {
	client := &http.Client{
		Timeout: 500 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	if total <= 0 {
		// unknown size → indeterminate progress
		fyne.Do(func() {
			progress.SetValue(0)
		})
	}

	pr := &progressReader{
		reader:   resp.Body,
		total:    total,
		progress: progress,
	}

	_, err = io.Copy(out, pr)
	return err
}

func replaceBinarySafe(tmpFile string, serverBinary string) error {

	// stop server first
	if soham_whatsapp_keys.ServerCmd != nil && soham_whatsapp_keys.ServerCmd.Process != nil {
		soham_whatsapp_keys.ServerCmd.Process.Kill()
		time.Sleep(3 * time.Second)
	}

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
