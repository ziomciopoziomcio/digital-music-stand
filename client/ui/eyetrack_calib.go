package ui

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ziomciopoziomcio/digital-music-stand/client/eyetrack"
)

func ShowEyetrackCalibration(w fyne.Window, app fyne.App) {
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

	var d dialog.Dialog
	wrapper := container.NewMax()

	camSelect := widget.NewSelect([]string{"Scanning..."}, nil)
	go func() {
		cams := eyetrack.GetAvailableCameras()
		camSelect.Options = cams
		if len(cams) > 0 {
			camSelect.SetSelected(cams[0])
		}
	}()

	var showStep2 func()
	var showStep3 func()

	step1 := container.NewVBox(
		widget.NewLabelWithStyle("Step 1: Select Camera", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		camSelect,
		widget.NewButton("Start Calibration", func() {
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
					_ = calibTracker.Start(cID, true)
					prog.Hide()
					showStep2()
				}()
			}
		}),
		widget.NewButton("Cancel", func() {
			close(stopChan)
			d.Hide()
		}),
	)

	showStep2 = func() {
		leftDot := canvas.NewCircle(theme.ErrorColor())
		sizedLeftDot := container.NewGridWrap(fyne.NewSize(60, 60), leftDot)

		content := container.NewBorder(
			widget.NewLabelWithStyle("Step 2: Look at the RED DOT and click NEXT", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewButton("NEXT", func() {
				app.Preferences().SetFloat("eyetrack_min_x", latestX)
				showStep3()
			}),
			container.NewCenter(sizedLeftDot),
			nil,
			container.NewCenter(imgCanvas),
		)
		wrapper.Objects = []fyne.CanvasObject{container.NewPadded(content)}
		wrapper.Refresh()
	}

	showStep3 = func() {
		rightDot := canvas.NewCircle(theme.ErrorColor())
		sizedRightDot := container.NewGridWrap(fyne.NewSize(60, 60), rightDot)

		content := container.NewBorder(
			widget.NewLabelWithStyle("Step 3: Look at the RED DOT and click FINISH", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewButton("FINISH", func() {
				app.Preferences().SetFloat("eyetrack_max_x", latestX)
				close(stopChan)
				d.Hide()
			}),
			nil,
			container.NewCenter(sizedRightDot),
			container.NewCenter(imgCanvas),
		)
		wrapper.Objects = []fyne.CanvasObject{container.NewPadded(content)}
		wrapper.Refresh()
	}

	wrapper.Objects = []fyne.CanvasObject{container.NewPadded(step1)}

	d = dialog.NewCustomWithoutButtons("Calibration", wrapper, w)
	d.Resize(fyne.NewSize(700, 500))
	d.Show()
}
