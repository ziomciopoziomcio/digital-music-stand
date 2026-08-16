package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/skip2/go-qrcode"

	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver"
)

func ShowPairingDialog(w fyne.Window, wsMgr *webserver.Manager) {
	ip := webserver.GetLocalIP()
	port := 8088
	url := fmt.Sprintf("http://%s:%d", ip, port)

	qrBytes, err := qrcode.Encode(url, qrcode.Medium, 256)
	var qrCanvas fyne.CanvasObject

	if err == nil {
		res := fyne.NewStaticResource("qr.png", qrBytes)
		img := canvas.NewImageFromResource(res)
		img.FillMode = canvas.ImageFillContain
		img.SetMinSize(fyne.NewSize(160, 160))
		qrCanvas = img
	} else {
		qrCanvas = widget.NewLabel("Failed to generate QR")
	}

	urlLabel := widget.NewLabelWithStyle(url, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	instructionLabel := widget.NewLabelWithStyle("1. Open URL in browser\n2. Enter the PIN shown on screen", fyne.TextAlignCenter, fyne.TextStyle{})

	pinEntry := widget.NewEntry()
	pinEntry.SetPlaceHolder("Enter 4-digit PIN")

	statusLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	var confirmBtn *widget.Button

	confirmBtn = widget.NewButtonWithIcon("Confirm PIN", theme.ConfirmIcon(), func() {
		if wsMgr.ConfirmPIN(pinEntry.Text) {
			statusLabel.SetText("Connected successfully! Ready for uploads.")
			statusLabel.TextStyle = fyne.TextStyle{Bold: true}
			pinEntry.Disable()
			confirmBtn.Disable()
		} else {
			statusLabel.SetText("Invalid PIN or browser not connected yet.")
			statusLabel.TextStyle = fyne.TextStyle{Italic: true}
			statusLabel.Refresh()
		}
	})
	confirmBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		container.NewCenter(qrCanvas),
		urlLabel,
		instructionLabel,
		widget.NewSeparator(),
		pinEntry,
		confirmBtn,
		statusLabel,
	)

	d := dialog.NewCustom("Device Pairing", "Close", content, w)
	d.Resize(fyne.NewSize(380, 450))
	d.Show()
}
