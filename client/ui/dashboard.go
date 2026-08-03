package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func BuildDashboard(w fyne.Window, app fyne.App, openSettings func(), openLogin func()) *fyne.Container {
	clock := widget.NewLabel(time.Now().Format("15:04"))
	clock.TextStyle = fyne.TextStyle{Bold: true}

	token := app.Preferences().String("jwt_token")
	statusText := "Offline"
	statusIcon := theme.WarningIcon()

	if token != "" {
		statusText = "Connected"
		statusIcon = theme.ConfirmIcon()
	}

	cloudStatusBtn := widget.NewButtonWithIcon(statusText, statusIcon, openLogin)
	if token == "" {
		cloudStatusBtn.Importance = widget.WarningImportance
	}

	settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), openSettings)

	topBar := container.NewHBox(
		clock,
		layout.NewSpacer(),
		cloudStatusBtn,
		settingsBtn,
	)

	practiceBtn := widget.NewButtonWithIcon("Practice Mode", theme.DocumentCreateIcon(), func() {})
	practiceBtn.Importance = widget.HighImportance

	concertBtn := widget.NewButtonWithIcon("Concert Mode", theme.MediaPlayIcon(), func() {})
	concertBtn.Importance = widget.HighImportance

	modesGrid := container.NewPadded(container.NewGridWithColumns(2,
		practiceBtn,
		concertBtn,
	))

	return container.NewBorder(topBar, nil, nil, nil, modesGrid)
}
