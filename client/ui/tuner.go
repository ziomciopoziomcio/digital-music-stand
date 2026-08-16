package ui

import (
	"math"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func ShowTunerDialog(w fyne.Window) {
	noteText := canvas.NewText("--", theme.ForegroundColor())
	noteText.TextSize = 120
	noteText.Alignment = fyne.TextAlignCenter
	noteText.TextStyle = fyne.TextStyle{Bold: true}

	centsBar := widget.NewProgressBar()
	centsBar.Min = -50
	centsBar.Max = 50
	centsBar.SetValue(0)

	statusLabel := widget.NewLabelWithStyle("Ready", fyne.TextAlignCenter, fyne.TextStyle{})

	var d dialog.Dialog
	listening := false
	var ticker *time.Ticker

	notes := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	updateTuner := func(freq float64) {
		if freq < 20 {
			noteText.Text = "--"
			centsBar.SetValue(0)
			statusLabel.SetText("No signal")
			noteText.Refresh()
			return
		}

		halfSteps := 12.0 * math.Log2(freq/440.0)
		nearestStep := math.Round(halfSteps)
		cents := (halfSteps - nearestStep) * 100.0

		noteIndex := (int(nearestStep) + 9) % 12
		if noteIndex < 0 {
			noteIndex += 12
		}

		noteText.Text = notes[noteIndex]
		noteText.Refresh()

		centsBar.SetValue(cents)

		if math.Abs(cents) < 5 {
			statusLabel.SetText("In Tune")
		} else if cents < 0 {
			statusLabel.SetText("Too Flat")
		} else {
			statusLabel.SetText("Too Sharp")
		}
	}

	toggleBtn := widget.NewButtonWithIcon("Start Listening", theme.MediaRecordIcon(), nil)
	toggleBtn.Importance = widget.HighImportance

	toggleBtn.OnTapped = func() {
		listening = !listening
		if listening {
			toggleBtn.SetText("Stop Listening")
			toggleBtn.SetIcon(theme.MediaStopIcon())
			ticker = time.NewTicker(100 * time.Millisecond)

			go func() {
				baseFreq := 440.0
				for range ticker.C {
					fluctuation := (rand.Float64() - 0.5) * 15.0
					updateTuner(baseFreq + fluctuation)
				}
			}()
		} else {
			toggleBtn.SetText("Start Listening")
			toggleBtn.SetIcon(theme.MediaRecordIcon())
			if ticker != nil {
				ticker.Stop()
			}
			updateTuner(0)
		}
	}

	closeBtn := widget.NewButton("Close", func() {
		if ticker != nil {
			ticker.Stop()
		}
		d.Hide()
	})

	content := container.NewVBox(
		container.NewCenter(noteText),
		widget.NewLabel(""),
		centsBar,
		statusLabel,
		widget.NewLabel(""),
		toggleBtn,
		closeBtn,
	)

	d = dialog.NewCustomWithoutButtons("Instrument Tuner", container.NewPadded(content), w)
	d.Show()
}
