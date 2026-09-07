package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
	"github.com/ziomciopoziomcio/digital-music-stand/client/remote"
	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver"
)

func ShowToolsMenu(w fyne.Window, metroAudio *audio.MetronomeAudio, remoteServer *remote.Server, setDialogBeatCb func(func(bool))) {
	var d dialog.Dialog

	tunerBtn := widget.NewButtonWithIcon("Tuner", theme.SettingsIcon(), func() {
		d.Hide()
		ShowTunerDialog(w)
	})
	tunerBtn.Importance = widget.HighImportance

	metronomeBtn := widget.NewButtonWithIcon("Metronome", theme.HistoryIcon(), func() {
		d.Hide()
		ShowMetronomeDialog(w, metroAudio, setDialogBeatCb)
	})
	metronomeBtn.Importance = widget.HighImportance

	pilotBtn := widget.NewButtonWithIcon("Pair Mobile Pilot", theme.ComputerIcon(), func() {
		if remoteServer == nil {
			dialog.ShowInformation("Error", "Remote server is not running.", w)
			return
		}

		ip := webserver.GetLocalIP()
		pin := remoteServer.GetPIN()

		msg := fmt.Sprintf("Enter these details in your mobile app:\n\nIP Address: %s\nPIN Code: %s", ip, pin)
		dialog.ShowInformation("Mobile Pilot Pairing", msg, w)
	})
	pilotBtn.Importance = widget.HighImportance

	closeBtn := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		d.Hide()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Practice Tools", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel(""),
		tunerBtn,
		widget.NewLabel(""),
		metronomeBtn,
		widget.NewLabel(""),
		pilotBtn,
		widget.NewLabel(""),
		widget.NewSeparator(),
		closeBtn,
	)

	d = dialog.NewCustomWithoutButtons("Tools", container.NewPadded(content), w)
	d.Show()
}
