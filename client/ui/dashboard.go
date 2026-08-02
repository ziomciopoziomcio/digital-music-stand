package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func BuildDashboard(w fyne.Window, openSettings func()) *fyne.Container {
	clock := widget.NewLabel(time.Now().Format("15:04"))
	clock.TextStyle = fyne.TextStyle{Bold: true}

	cloudStatusBtn := widget.NewButtonWithIcon("Offline", theme.WarningIcon(), func() {})
	cloudStatusBtn.Importance = widget.WarningImportance

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
