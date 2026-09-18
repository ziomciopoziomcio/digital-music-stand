package ui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ziomciopoziomcio/digital-music-stand/client/eyetrack"
)

func ShowEyetrackCalibration(w fyne.Window, app fyne.App) {
	enabled := app.Preferences().BoolWithFallback("eyetrack_enabled", false)
	camID := app.Preferences().IntWithFallback("eyetrack_camera", 0)

	camEntry := widget.NewEntry()
	camEntry.SetText(fmt.Sprintf("%d", camID))

	imgCanvas := canvas.NewImageFromResource(nil)
	imgCanvas.FillMode = canvas.ImageFillContain
	imgCanvas.SetMinSize(fyne.NewSize(320, 240))

	calibTracker := eyetrack.NewTracker()
	stopChan := make(chan struct{})

	startPreview := func() {
		calibTracker.Stop()
		cID, _ := strconv.Atoi(camEntry.Text)
		app.Preferences().SetInt("eyetrack_camera", cID)

		prog := dialog.NewCustomWithoutButtons("Eye Tracker", container.NewPadded(container.NewVBox(
			widget.NewLabelWithStyle("Initializing AI models...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("This might take a minute on the first run."),
			widget.NewProgressBarInfinite(),
		)), w)
		prog.Show()

		go func() {
			err := calibTracker.Start(cID, true)
			prog.Hide()
			if err != nil {
				dialog.ShowError(err, w)
			}
		}()
	}

	if enabled {
		startPreview()
	}

	go func() {
		for {
			select {
			case <-stopChan:
				calibTracker.Stop()
				return
			case pt := <-calibTracker.GazeChan:
				if pt.Frame != "" {
					data, err := base64.StdEncoding.DecodeString(pt.Frame)
					if err == nil {
						img, _, err := image.Decode(bytes.NewReader(data))
						if err == nil {
							imgCanvas.Image = img
							imgCanvas.Refresh()
						}
					}
				}
			}
		}
	}()

	var d dialog.Dialog
	applyBtn := widget.NewButtonWithIcon("Apply Camera", theme.ViewRefreshIcon(), func() {
		startPreview()
	})
	closeBtn := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		close(stopChan)
		calibTracker.Stop()
		d.Hide()
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Eye Tracking Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewHBox(widget.NewLabel("Camera Index:"), camEntry, applyBtn),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Camera Preview", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewCenter(imgCanvas),
		widget.NewLabelWithStyle("Keep your head center. Look left/right.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		widget.NewSeparator(),
		closeBtn,
	)

	d = dialog.NewCustomWithoutButtons("Eye Tracking Calibration", container.NewPadded(form), w)
	d.Show()
}
