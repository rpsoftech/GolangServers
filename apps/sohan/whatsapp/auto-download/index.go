package soham_whatsapp_auto_download

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	soham_whatsapp_keys "github.com/rpsoftech/golang-servers/apps/sohan/whatsapp/keys"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	"github.com/rpsoftech/golang-servers/utility/updater"
)

var checkAndRunCalled = false

// CheckAndDownload keeps the WhatsApp client binary in ConfigDir on the newest
// signed release and returns its path. The progress window is only shown when
// a download actually happens. It panics only when no client is installed and
// none could be downloaded, because the launcher has nothing to start then.
func CheckAndDownload(progress *widget.ProgressBar, win fyne.Window) string {
	serverBinary := "whatsapp-client.o"
	if runtime.GOOS == "windows" {
		serverBinary = "whatsapp-client.exe"
	}
	serverBinary = filepath.Join(soham_whatsapp_keys.ConfigDir, serverBinary)
	if checkAndRunCalled {
		return serverBinary
	}

	// The launcher is not built by the release pipeline, so it follows
	// PRODUCTION unless it was built with an explicit updater.BuildEnv.
	envName := updater.BuildEnv
	if envName == "" {
		envName = "PRODUCTION"
	}

	var showOnce sync.Once
	cfg := updater.Config{
		Component: updater.SohamWhatsappClientProjectName,
		Env:       envName,
		OnProgress: func(read, total int64) {
			showOnce.Do(func() { fyne.DoAndWait(func() { win.Show() }) })
			if total > 0 {
				value := float64(read) / float64(total)
				fyne.Do(func() { progress.SetValue(value) })
			}
		},
		// Stop the running client before its binary is swapped.
		BeforeReplace: func() {
			if soham_whatsapp_keys.ServerCmd != nil && soham_whatsapp_keys.ServerCmd.Process != nil {
				soham_whatsapp_keys.ServerCmd.Process.Kill()
				time.Sleep(3 * time.Second)
			}
		},
	}

	if _, err := updater.UpdateFile(context.Background(), cfg, serverBinary); err != nil {
		log.Printf("Client update check failed: %v", err)
		if exist, _ := utility_functions.Exist(serverBinary); !exist {
			panic(fmt.Errorf("no WhatsApp client installed and download failed: %w", err))
		}
	}
	checkAndRunCalled = true
	return serverBinary
}
