package ui

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ziomciopoziomcio/digital-music-stand/client/eyetrack"
)

func ShowEyetrackCalibration(w fyne.Window, app fyne.App) {
	oldContent := w.Content()
	wasFullScreen := w.FullScreen()

	calibTracker := eyetrack.GetTracker()
	stopChan := make(chan struct{})
	var latestX float64

	imgCanvas := canvas.NewImageFromResource(nil)
	imgCanvas.FillMode = canvas.ImageFillContain
	imgCanvas.SetMinSize(fyne.NewSize(400, 300))

	go func() {
		for {
			select {
			case <-stopChan:
				calibTracker.Stop()
				return
			case pt := <-calibTracker.GazeChan:
				latestX = pt.X
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

	var stopOnce sync.Once
	cleanup := func() {
		stopOnce.Do(func() {
			close(stopChan)
		})
		calibTracker.Stop()
		w.SetContent(oldContent)
		w.SetFullScreen(wasFullScreen)
	}

	camSelect := widget.NewSelect([]string{"Scanning..."}, nil)
	go func() {
		cams := eyetrack.GetAvailableCameras()
		camSelect.Options = cams
		if len(cams) > 0 {
			camSelect.SetSelected(cams[0])
		}
		camSelect.Refresh()
	}()

	leftDot := canvas.NewCircle(theme.ErrorColor())
	sizedLeftDot := container.NewGridWrap(fyne.NewSize(60, 60), leftDot)
	leftEdge := container.NewVBox(layout.NewSpacer(), sizedLeftDot, layout.NewSpacer())

	rightDot := canvas.NewCircle(theme.ErrorColor())
	sizedRightDot := container.NewGridWrap(fyne.NewSize(60, 60), rightDot)
	rightEdge := container.NewVBox(layout.NewSpacer(), sizedRightDot, layout.NewSpacer())

	var step2, step3 *fyne.Container

	btnNext := widget.NewButton("NEXT", func() {
		app.Preferences().SetFloat("eyetrack_min_x", latestX)
		w.SetContent(step3)
	})
	btnCancel2 := widget.NewButton("Cancel", cleanup)

	centerPanel2 := container.NewCenter(
		container.NewVBox(
			widget.NewLabelWithStyle("Step 2: Look at the RED DOT on the left and click NEXT", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			btnNext,
			imgCanvas,
			btnCancel2,
		),
	)
	step2 = container.NewPadded(container.NewBorder(nil, nil, leftEdge, nil, centerPanel2))

	btnFinish := widget.NewButton("FINISH", func() {
		app.Preferences().SetFloat("eyetrack_max_x", latestX)
		cleanup()
	})
	btnCancel3 := widget.NewButton("Cancel", cleanup)

	centerPanel3 := container.NewCenter(
		container.NewVBox(
			widget.NewLabelWithStyle("Step 3: Look at the RED DOT on the right and click FINISH", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			btnFinish,
			imgCanvas,
			btnCancel3,
		),
	)
	step3 = container.NewPadded(container.NewBorder(nil, nil, nil, rightEdge, centerPanel3))

	btnStart := widget.NewButton("Start Calibration", func() {
		if camSelect.Selected == "" || camSelect.Selected == "Scanning..." {
			return
		}
		parts := strings.Split(camSelect.Selected, " ")
		if len(parts) == 2 {
			cID, _ := strconv.Atoi(parts[1])
			app.Preferences().SetInt("eyetrack_camera", cID)

			prog := dialog.NewCustomWithoutButtons("Starting...", container.NewPadded(widget.NewProgressBarInfinite()), w)
			prog.Show()

			go func() {
				err := calibTracker.Start(cID, true)
				prog.Hide()
				if err != nil {
					dialog.ShowError(err, w)
				} else {
					w.SetFullScreen(true)
					w.SetContent(step2)
				}
			}()
		}
	})
	btnCancel1 := widget.NewButton("Cancel", cleanup)

	step1 := container.NewPadded(container.NewCenter(
		container.NewVBox(
			widget.NewLabelWithStyle("Step 1: Select Camera", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			camSelect,
			btnStart,
			btnCancel1,
		),
	))

	w.SetContent(step1)
}
