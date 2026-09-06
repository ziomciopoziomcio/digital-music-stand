package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
)

var AppVersion = "mobile-v0.1.0-alpha.1"

func main() {
	myApp := app.NewWithID("com.digitalmusicstand.pilot")
	myWindow := myApp.NewWindow("DMS Pilot")

	myWindow.Resize(fyne.NewSize(360, 800))

	var showLogin func()
	var showPilot func(server, token, concertID string)

	showLogin = func() {
		serverEntry := widget.NewEntry()
		serverEntry.SetPlaceHolder("localhost:50051")
		serverEntry.SetText(myApp.Preferences().StringWithFallback("server", "localhost:50051"))

		tokenEntry := widget.NewPasswordEntry()
		tokenEntry.SetPlaceHolder("JWT Token")
		tokenEntry.SetText(myApp.Preferences().String("token"))

		concertEntry := widget.NewEntry()
		concertEntry.SetPlaceHolder("Concert ID")
		concertEntry.SetText(myApp.Preferences().String("concert_id"))

		connectBtn := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() {
			myApp.Preferences().SetString("server", serverEntry.Text)
			myApp.Preferences().SetString("token", tokenEntry.Text)
			myApp.Preferences().SetString("concert_id", concertEntry.Text)

			showPilot(serverEntry.Text, tokenEntry.Text, concertEntry.Text)
		})
		connectBtn.Importance = widget.HighImportance

		form := container.NewVBox(
			widget.NewLabelWithStyle("DMS Pilot", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithStyle(AppVersion, fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
			widget.NewSeparator(),
			widget.NewLabel("Server Address:"), serverEntry,
			widget.NewLabel("Access Token:"), tokenEntry,
			widget.NewLabel("Concert ID:"), concertEntry,
			widget.NewLabel(""),
			connectBtn,
		)

		myWindow.SetContent(container.NewPadded(form))
	}

	showPilot = func(server, token, concertID string) {
		statusLabel := widget.NewLabelWithStyle("Status: Connecting...", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

		sendCommand := func(action syncpb.ActionType) {
			// TODO: Sending commands via gRPC stream
			// syncStream.Send(&syncpb.SyncRequest{...})
		}

		prevItemBtn := widget.NewButtonWithIcon("Prev Item", theme.MediaSkipPreviousIcon(), func() {
			sendCommand(syncpb.ActionType_PREV_ITEM)
		})
		nextItemBtn := widget.NewButtonWithIcon("Next Item", theme.MediaSkipNextIcon(), func() {
			sendCommand(syncpb.ActionType_NEXT_ITEM)
		})
		prevPageBtn := widget.NewButtonWithIcon("Prev Page", theme.NavigateBackIcon(), func() {
			sendCommand(syncpb.ActionType_PREV_PAGE)
		})
		nextPageBtn := widget.NewButtonWithIcon("Next Page", theme.NavigateNextIcon(), func() {
			sendCommand(syncpb.ActionType_NEXT_PAGE)
		})
		timerBtn := widget.NewButtonWithIcon("Toggle Timer", theme.HistoryIcon(), func() {
			sendCommand(syncpb.ActionType_TOGGLE_TIMER)
		})

		prevItemBtn.Importance = widget.HighImportance
		nextItemBtn.Importance = widget.HighImportance

		grid := container.NewGridWithColumns(2,
			prevItemBtn, nextItemBtn,
			prevPageBtn, nextPageBtn,
		)

		disconnectBtn := widget.NewButtonWithIcon("Disconnect", theme.CancelIcon(), func() {
			showLogin()
		})
		disconnectBtn.Importance = widget.DangerImportance

		layoutWrapper := container.NewVBox(
			widget.NewLabelWithStyle("Remote Control", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			statusLabel,
			widget.NewSeparator(),
			widget.NewLabel(""),
			grid,
			widget.NewLabel(""),
			timerBtn,
			layout.NewSpacer(),
			widget.NewSeparator(),
			disconnectBtn,
		)

		myWindow.SetContent(container.NewPadded(layoutWrapper))

		// TODO: stream connection via gRPC
	}

	showLogin()
	myWindow.ShowAndRun()
}
