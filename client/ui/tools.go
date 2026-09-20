package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/remote"
	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver"
)

func ShowToolsMenu(w fyne.Window, app fyne.App, metroAudio *audio.MetronomeAudio, recorderAudio *audio.RecorderAudio, remoteServer *remote.Server, db *localdb.DBManager, scoreID, scoreTitle, profilePath string, setDialogBeatCb func(func(bool))) {
	var d dialog.Dialog

	tunerBtn := NewTouchButtonWithIcon("Tuner", theme.SettingsIcon(), func() {
		d.Hide()
		ShowTunerDialog(w)
	})
	tunerBtn.Importance = widget.HighImportance

	metronomeBtn := NewTouchButtonWithIcon("Metronome", theme.HistoryIcon(), func() {
		d.Hide()
		ShowMetronomeDialog(w, metroAudio, setDialogBeatCb)
	})
	metronomeBtn.Importance = widget.HighImportance

	dictaphoneBtn := NewTouchButtonWithIcon("Dictaphone", theme.MediaRecordIcon(), func() {
		d.Hide()
		ShowDictaphoneDialog(w, db, recorderAudio, scoreID, scoreTitle, profilePath)
	})
	dictaphoneBtn.Importance = widget.HighImportance

	pilotBtn := NewTouchButtonWithIcon("Pair Mobile Pilot", theme.ComputerIcon(), func() {
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

	p2pBtn := NewTouchButtonWithIcon("P2P Network Status", theme.InfoIcon(), func() {
		if remoteServer == nil {
			return
		}
		peers := remoteServer.GetDiscoveredPeers()
		var items []fyne.CanvasObject
		for _, p := range peers {
			items = append(items, widget.NewCard(p.ConcertID, fmt.Sprintf("IP: %s:%d\nLast Seen: %s", p.IP, p.Port, p.LastSeen.Format("15:04:05")), nil))
		}
		if len(items) == 0 {
			items = append(items, widget.NewLabelWithStyle("No active P2P leaders discovered nearby.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}))
		}
		d.Hide()
		dialog.ShowCustom("Discovered Devices (LAN)", "Close", container.NewVScroll(container.NewVBox(items...)), w)
	})
	p2pBtn.Importance = widget.WarningImportance

	closeBtn := NewTouchButtonWithIcon("Close", theme.CancelIcon(), func() {
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
		dictaphoneBtn,
		widget.NewLabel(""),
		pilotBtn,
		widget.NewLabel(""),
		p2pBtn,
		widget.NewLabel(""),
		widget.NewSeparator(),
		closeBtn,
	)

	d = dialog.NewCustomWithoutButtons("Tools", container.NewPadded(content), w)
	d.Show()
}
