//go:build !linux

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func main() {

	// Create app
	myApp := app.New()
	// myApp.Settings().SetTheme(th) // Professional dark UI

	window := myApp.NewWindow("Whatsapp API(WBOT)")
	iconResource, _ := fyne.LoadResourceFromPath("icon.png")
	window.SetIcon(iconResource)
	window.Resize(fyne.NewSize(600, 420))
	window.CenterOnScreen()
	// ================= SYSTEM TRAY =================

	if desk, ok := myApp.(desktop.App); ok {

		showItem := fyne.NewMenuItem("Show", func() {
			window.Show()
			window.RequestFocus()
		})

		quitItem := fyne.NewMenuItem("Quit", func() {
			myApp.Quit()
		})

		trayMenu := fyne.NewMenu("RP Softech",
			showItem,
			fyne.NewMenuItemSeparator(),
			quitItem,
		)

		desk.SetSystemTrayMenu(trayMenu)

		// clicking tray icon restores window
		desk.SetSystemTrayIcon(iconResource)
	}

	// ================= LOGO IMAGE =================

	logoURI, _ := storage.ParseURI("file://icon.png")
	logoImage := canvas.NewImageFromURI(logoURI)

	logoImage.FillMode = canvas.ImageFillContain
	logoImage.SetMinSize(fyne.NewSize(80, 80))

	logoContainer := container.NewHBox(
		layout.NewSpacer(),
		logoImage,
		layout.NewSpacer(),
	)

	// ================= TITLE =================

	title := widget.NewLabelWithStyle(
		"WABOT UTILITY",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	title.TextStyle = fyne.TextStyle{Bold: true}

	titleBox := container.NewPadded(
		container.NewHBox(
			layout.NewSpacer(),
			title,
			layout.NewSpacer(),
		),
	)
	centeredCard := ShowInputDetails()
	// ================= MAIN LAYOUT =================

	mainLayout := container.NewVBox(
		logoContainer,
		titleBox,
		layout.NewSpacer(),
		centeredCard,
		layout.NewSpacer(),
	)
	window.SetCloseIntercept(func() {
		window.Hide()
	})
	window.SetContent(mainLayout)
	window.ShowAndRun()
}

func ShowInputDetails() *fyne.Container {
	// ================= INPUT 1 =================

	entry1 := widget.NewEntry()
	entry1.SetPlaceHolder("Enter First Value")
	entry1.Resize(fyne.NewSize(300, 35))

	formRow1 := container.NewGridWithColumns(2,
		widget.NewLabel("Field 1:"),
		entry1,
	)

	// ================= INPUT 2 =================

	entry2 := widget.NewEntry()
	entry2.SetPlaceHolder("Enter Second Value")

	formRow2 := container.NewGridWithColumns(2,
		widget.NewLabel("Field 2:"),
		entry2,
	)

	// ================= STATUS LABEL =================

	status := widget.NewLabel("Ready")
	status.Alignment = fyne.TextAlignCenter
	status.TextStyle = fyne.TextStyle{Bold: true}

	// ================= BUTTONS =================

	btnStart := widget.NewButton("Start", func() {
		status.SetText("Start button clicked")
	})
	buttonRow := container.NewHBox(
		layout.NewSpacer(),
		btnStart,
		layout.NewSpacer(),
	)

	// ================= FORM CARD (Panel Look) =================

	formContent := container.NewVBox(
		layout.NewSpacer(),
		formRow1,
		formRow2,
		layout.NewSpacer(),
		buttonRow,
		layout.NewSpacer(),
		status,
		layout.NewSpacer(),
	)

	card := widget.NewCard(
		"Control Panel",
		"Enter details and choose action",
		container.NewPadded(formContent),
	)

	// Center the card in window
	return container.NewHBox(
		layout.NewSpacer(),
		container.NewStack(card),
		layout.NewSpacer(),
	)
}
