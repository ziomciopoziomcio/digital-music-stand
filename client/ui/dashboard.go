package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func BuildDashboard(w fyne.Window, app fyne.App, openSettings func(), openLogin func(), openPractice func(), openConcert func(), openPairing func(), openInbox func(), openProfile func()) *fyne.Container {
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

	inboxBtn := widget.NewButtonWithIcon("Inbox", theme.InfoIcon(), openInbox)
	profileBtn := widget.NewButtonWithIcon("Profile", theme.AccountIcon(), openProfile)
	settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), openSettings)

	topBar := container.NewHBox(
		clock,
		layout.NewSpacer(),
		inboxBtn,
		profileBtn,
		cloudStatusBtn,
		settingsBtn,
	)

	practiceBtn := widget.NewButtonWithIcon("Practice Mode", theme.DocumentCreateIcon(), openPractice)
	practiceBtn.Importance = widget.HighImportance

	concertBtn := widget.NewButtonWithIcon("Concert Mode", theme.MediaPlayIcon(), openConcert)
	concertBtn.Importance = widget.HighImportance

	pairingBtn := widget.NewButtonWithIcon("Pair Remote", theme.ComputerIcon(), openPairing)

	grid := container.NewGridWithColumns(2, practiceBtn, concertBtn)

	bottomArea := container.NewVBox(
		widget.NewSeparator(),
		pairingBtn,
	)

	mainContent := container.NewBorder(
		nil,
		bottomArea,
		nil,
		nil,
		grid,
	)

	return container.NewBorder(
		container.NewPadded(topBar),
		nil, nil, nil,
		container.NewPadded(mainContent),
	)
}
