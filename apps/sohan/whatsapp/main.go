package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	sohan_whatsapp_auto_download "github.com/rpsoftech/golang-servers/apps/sohan/whatsapp/auto-download"
	soham_whatsapp_gui_config "github.com/rpsoftech/golang-servers/apps/sohan/whatsapp/config"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_interfaces "github.com/rpsoftech/golang-servers/interfaces/whatsapp"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

//////////////////////////////////////////////////////
// GLOBALS
//////////////////////////////////////////////////////

var mainWindow fyne.Window
var trayStatusItem *fyne.MenuItem
var trayMenu *fyne.Menu
var qrContainer = container.NewCenter(
	widget.NewLabel("Loading QR..."),
)
var version string
var configFile = "config.json"
var serverLock = "server.lock"
var guiLock = "gui.lock"

//////////////////////////////////////////////////////
// CONFIG STRUCT
//////////////////////////////////////////////////////

type Config struct {
	Token  string `json:"token"`
	Number string `json:"number"`
}

//////////////////////////////////////////////////////
// SINGLE GUI INSTANCE
//////////////////////////////////////////////////////

func validateGUILOCK() bool {
	data, err := os.ReadFile(guiLock)

	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(data))

	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)

	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil

}
func preventMultipleGUI() {

	if _, err := os.Stat(guiLock); err == nil {
		if validateGUILOCK() {
			os.Exit(0)
		}
	}

	os.WriteFile(guiLock, []byte(strconv.Itoa(os.Getpid())), 0644)
}

//////////////////////////////////////////////////////
// SERVER INSTANCE CHECK
//////////////////////////////////////////////////////

func serverRunning() bool {

	data, err := os.ReadFile(serverLock)

	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(data))

	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)

	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))

	return err == nil
}

//////////////////////////////////////////////////////
// START SERVER
//////////////////////////////////////////////////////

func startServer(a fyne.App) {
	if serverRunning() {
		return
	}
	progressBar, progressWin := showDownloadProgress(a)

	go func() {
		serverBinary := sohan_whatsapp_auto_download.CheckAndDownload(progressBar, progressWin)
		// start server
		fyne.Do(func() {
			progressWin.Close()
		})
		for {
			updateTrayStatus("🟡 Starting server...")
			soham_whatsapp_gui_config.ServerCmd = exec.Command(filepath.Join(env.FindAndReturnCurrentDir(), serverBinary))
			serverCmd := soham_whatsapp_gui_config.ServerCmd
			if env.IsDev {
				stdout, err := serverCmd.StdoutPipe()
				if err != nil {
					log.Println("stdout pipe error:", err)
					time.Sleep(3 * time.Second)
					continue
				}

				stderr, err := serverCmd.StderrPipe()
				if err != nil {
					log.Println("stderr pipe error:", err)
					time.Sleep(3 * time.Second)
					continue
				}
				go utility_functions.StreamLogs(stdout, "SERVER")
				go utility_functions.StreamLogs(stderr, "SERVER")
			}
			err := soham_whatsapp_gui_config.ServerCmd.Start()
			if err != nil {
				updateTrayStatus("🔴 Server failed")
				time.Sleep(50 * time.Second)
				continue
			}

			updateTrayStatus("🟢 Server running")
			err = soham_whatsapp_gui_config.ServerCmd.Wait()
			updateTrayStatus("🔴 Server stopped")
			time.Sleep(3 * time.Second)

		}
	}()
}

//////////////////////////////////////////////////////
//// Stream Logs /////////////////////////////////////
//////////////////////////////////////////////////////

//////////////////////////////////////////////////////
// RESTART SERVER
//////////////////////////////////////////////////////

func restartServer() {

	if soham_whatsapp_gui_config.ServerCmd != nil && soham_whatsapp_gui_config.ServerCmd.Process != nil {
		soham_whatsapp_gui_config.ServerCmd.Process.Kill()
	}

}

//////////////////////////////////////////////////////
// SAVE CONFIG
//////////////////////////////////////////////////////

func saveConfig(token, number string) {
	config := new(whatsapp_interfaces.IServerConfig)
	config.Tokens = make(map[string]string)
	config.JID = make(map[string]string)
	config.Tokens[token] = number
	config.SetConfigPath(soham_whatsapp_gui_config.ServerConfigFilePath)
	soham_whatsapp_gui_config.SaveConfig(config)
}

//////////////////////////////////////////////////////
// LOAD CONFIG
//////////////////////////////////////////////////////

func loadConfig() bool {
	ok, _ := soham_whatsapp_gui_config.ValidateConfig()
	return ok
}

//////////////////////////////////////////////////////
// SUCCESS SCREEN
//////////////////////////////////////////////////////

func successScreen() fyne.CanvasObject {
	ok, config := soham_whatsapp_gui_config.ValidateConfig()
	if !ok {
		panic("Config issue")
	}
	initialLoggedin := true
	for token := range config.Tokens {
		go func() {
			time.Sleep(3 * time.Second)
			for {
				loggedin, err := soham_whatsapp_gui_config.LoginApiCall(token)
				if err != nil {
					fmt.Println(err)
					time.Sleep(5 * time.Second)
					continue
				}
				if loggedin {
					if initialLoggedin {
						fyne.Do(func() {
							mainWindow.Hide()
							initialLoggedin = false
						})
					}
					if !qrContainer.Hidden {
						fyne.Do(func() {
							qrContainer.Hide()
						})
					}
					time.Sleep(5 * time.Second)
					continue
				}
				ok, qrcode := soham_whatsapp_gui_config.QrCodeApiCall(token)
				if !ok {
					time.Sleep(5 * time.Second)
					continue
				}
				img, err := soham_whatsapp_gui_config.CreateQRFromBase64(qrcode)
				if err != nil {
					fmt.Println(err)
					time.Sleep(5 * time.Second)
					continue
				}
				fyne.DoAndWait(func() {
					if qrContainer.Hidden {
						qrContainer.Show()
					}
					// if mainWindow.
					qrContainer.Objects = []fyne.CanvasObject{
						container.NewCenter(img),
					}
					qrContainer.Refresh()

				})
				time.Sleep(10 * time.Second)
				// break
			}
		}()
	}
	msg := widget.NewLabelWithStyle(
		"Configuration Successful",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	restartBtn := widget.NewButton("Restart Server", func() {
		msg.SetText("Restarting Server...")
		restartServer()
	})

	// resetBtn := widget.NewButton("Reset Config", func() {

	// 	os.Remove(configFile)
	// 	mainWindow.SetContent(configForm(a))
	// })

	buttons := container.NewHBox(
		layout.NewSpacer(),
		restartBtn,
		// resetBtn,
		layout.NewSpacer(),
	)

	return container.NewVBox(
		layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), msg, layout.NewSpacer()),
		qrContainer,
		buttons,
		layout.NewSpacer(),
	)
}

//////////////////////////////////////////////////////
// CONFIG FORM
//////////////////////////////////////////////////////

func configForm(a fyne.App) fyne.CanvasObject {

	token := widget.NewEntry()
	number := widget.NewEntry()

	token.SetPlaceHolder("UUID Token")
	number.SetPlaceHolder("Phone Number")

	status := widget.NewLabel("")

	saveBtn := widget.NewButton("Save Config", func() {

		if ok, t := soham_whatsapp_gui_config.ValidUUID(token.Text); !ok {
			status.SetText(t)
			return
		}

		if number.Text == "" {
			status.SetText("Number required")
			return
		}

		saveConfig(token.Text, number.Text)

		startServer(a)

		mainWindow.SetContent(successScreen())

		go func() {
			time.Sleep(3 * time.Second)
			fyne.Do(func() {
				mainWindow.Hide()
			})

		}()
	})

	row1 := container.NewGridWithColumns(2,
		widget.NewLabel("Token"),
		token,
	)

	row2 := container.NewGridWithColumns(2,
		widget.NewLabel("Number"),
		number,
	)

	// embedded logo
	logo := canvas.NewImageFromFile("icon.png")

	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(120, 80))

	logoBox := container.NewHBox(
		layout.NewSpacer(),
		logo,
		layout.NewSpacer(),
	)

	return container.NewVBox(
		logoBox,
		layout.NewSpacer(),
		row1,
		row2,
		layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), saveBtn, layout.NewSpacer()),
		layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), status, layout.NewSpacer()),
		layout.NewSpacer(),
	)
}

func showDownloadProgress(a fyne.App) (*widget.ProgressBar, fyne.Window) {

	var win fyne.Window
	fyne.DoAndWait(func() {
		win = a.NewWindow("Downloading Update")
	})

	progress := widget.NewProgressBar()

	progress.SetValue(0)

	fyne.DoAndWait(func() {
		// win = a.NewWindow("Downloading Update")
		win.SetContent(
			container.NewVBox(
				widget.NewLabel("Downloading server update..."),
				progress,
			),
		)
		win.Resize(fyne.NewSize(400, 120))
		win.CenterOnScreen()
	})

	return progress, win
}

//////////////////////////////////////////////////////
// Update Tray Status
//////////////////////////////////////////////////////

func updateTrayStatus(status string) {

	if trayStatusItem == nil {
		return
	}

	fyne.DoAndWait(func() {
		log.Println("Here", status)
		trayStatusItem.Label = status
		// time.Sleep(1 * time.Second)
		trayMenu.Refresh()
	})
}

// ////////////////////////////////////////////////////
// SET Env For Client
// ////////////////////////////////////////////////////
// APP_ENV="LOCAL"
// PORT="4000"
// SOCKET_URL="ws://localhost:4001/whatsapp-client/ws"
// ALLOW_LOCAL_NO_AUTH=false
// AUTO_CONNECT_TO_WHATSAPP=true
// OPEN_BROWSER_FOR_SCAN=true
func SetEnv() {
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "4000")
	}
	if os.Getenv("SOCKET_URL") == "" {
		os.Setenv("SOCKET_URL", "ws://localhost:4001/whatsapp-client/ws")
	}
	if os.Getenv("ALLOW_LOCAL_NO_AUTH") == "" {
		os.Setenv("ALLOW_LOCAL_NO_AUTH", "false")
	}
	if os.Getenv("AUTO_CONNECT_TO_WHATSAPP") == "" {
		os.Setenv("AUTO_CONNECT_TO_WHATSAPP", "true")
	}
	if os.Getenv("OPEN_BROWSER_FOR_SCAN") == "" {
		os.Setenv("OPEN_BROWSER_FOR_SCAN", "false")
	}
}

//////////////////////////////////////////////////////
// MAIN
//////////////////////////////////////////////////////

func main() {
	if env.IsDev {
		version = "dev"
	}
	log.Printf("Starting WABOT GUI Version %s", version)
	if os.Getenv("APP_ENV") == "" {
		os.Setenv("APP_ENV", "PRODUCTION")
	}
	env.LoadEnv(filepath.Join(env.FindAndReturnCurrentDir(), "whatsapp-client.env"))
	SetEnv()
	log.Printf("ENV Settled")
	preventMultipleGUI()
	defer os.Remove(guiLock)
	a := app.New()
	// a.Settings().SetTheme(theme.)

	window := a.NewWindow("RP Softech Config Tool")
	window.Resize(fyne.NewSize(520, 380))
	window.CenterOnScreen()

	mainWindow = window

	// embedded icon
	iconResource, err := fyne.LoadResourceFromPath("icon.png")
	if err == nil {
		window.SetIcon(iconResource)
	}
	// window.SetIcon(icon)

	//////////////////////////////////////////////////
	// SYSTEM TRAY
	//////////////////////////////////////////////////

	if desk, ok := a.(desktop.App); ok {

		trayStatusItem = fyne.NewMenuItem("🟡 Starting...", nil)

		showItem := fyne.NewMenuItem("Open Config", func() {
			window.Show()
			window.RequestFocus()
		})
		restartItem := fyne.NewMenuItem("Restart Server", func() {
			restartServer()
		})
		quitItem := fyne.NewMenuItem("Quit", func() {
			os.Remove(guiLock)
			a.Quit()
		})

		trayMenu := fyne.NewMenu("WABOT Utility",
			trayStatusItem,
			fyne.NewMenuItemSeparator(),
			showItem,
			restartItem,
			fyne.NewMenuItemSeparator(),
			quitItem,
		)
		desk.SetSystemTrayMenu(trayMenu)
		desk.SetSystemTrayIcon(iconResource)
	}

	window.SetCloseIntercept(func() {
		window.Hide()
	})

	//////////////////////////////////////////////////
	// START LOGIC
	//////////////////////////////////////////////////

	if loadConfig() {
		window.SetContent(successScreen())
		go func() {
			startServer(a)
			time.Sleep(3 * time.Second)
			fyne.Do(func() {
				// window.Hide()
			})
		}()
	} else {
		window.SetContent(configForm(a))
	}
	window.Show()

	a.Run()
}
