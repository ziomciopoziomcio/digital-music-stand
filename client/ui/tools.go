package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
)

func ShowToolsMenu(w fyne.Window, metroAudio *audio.MetronomeAudio, setDialogBeatCb func(func(bool))) {
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
		widget.NewSeparator(),
		closeBtn,
	)

	d = dialog.NewCustomWithoutButtons("Tools", container.NewPadded(content), w)
	d.Show()
}
