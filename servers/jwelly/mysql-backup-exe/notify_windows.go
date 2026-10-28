//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

// Windows PowerShell's own AppUserModelID; toasts are only shown for a
// registered app id, and this one exists on every Windows install.
const toastAppID = `{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe`

// The text travels through environment variables so it can never be parsed as
// PowerShell code.
const toastScript = `
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml('<toast><visual><binding template="ToastGeneric"><text id="1"></text><text id="2"></text></binding></visual></toast>')
$t = $xml.GetElementsByTagName('text')
$t.Item(0).InnerText = $env:BACKUP_TOAST_TITLE
$t.Item(1).InnerText = $env:BACKUP_TOAST_BODY
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($env:BACKUP_TOAST_APPID).Show($toast)
`

type toastNotifier struct{}

func newNotifier() Notifier { return toastNotifier{} }

func (toastNotifier) Notify(title, message string) {
	log.Printf("NOTICE: %s: %s", title, message)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", toastScript)
	cmd.Env = append(os.Environ(),
		"BACKUP_TOAST_TITLE="+title,
		"BACKUP_TOAST_BODY="+message,
		"BACKUP_TOAST_APPID="+toastAppID,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("toast notification failed: %v %s", err, out)
	}
}

// ownsConsole is true when this process is the only one attached to its
// console, i.e. it was started by double-click and the window closes on exit.
func ownsConsole() bool {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleProcessList")
	var pids [2]uint32
	n, _, _ := proc.Call(uintptr(unsafe.Pointer(&pids[0])), 2)
	return n == 1
}
