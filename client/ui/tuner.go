package ui

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type meterLayout struct{}

func (m *meterLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(300, 140)
}

func (m *meterLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
}

func ShowTunerDialog(w fyne.Window) {
	noteText := canvas.NewText("--", theme.ForegroundColor())
	noteText.TextSize = 160
	noteText.Alignment = fyne.TextAlignCenter
	noteText.TextStyle = fyne.TextStyle{Bold: true}

	currentFreqLabel := widget.NewLabelWithStyle("Current: -- Hz", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	targetFreqLabel := widget.NewLabelWithStyle("Target: -- Hz", fyne.TextAlignCenter, fyne.TextStyle{})

	statusLabel := widget.NewLabelWithStyle("Ready", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	pivotX, pivotY := float32(150), float32(130)
	needleLength := float32(120)

	needle := canvas.NewLine(theme.DisabledColor())
	needle.StrokeWidth = 5

	meterObjects := []fyne.CanvasObject{}

	for i := -50; i <= 50; i += 5 {
		angle := (float64(i) / 50.0) * (math.Pi / 3.0)
		sin := float32(math.Sin(angle))
		cos := float32(math.Cos(angle))

		tickLength := float32(10)
		if i == 0 {
			tickLength = 25
		} else if i%10 == 0 {
			tickLength = 15
		}

		tick := canvas.NewLine(theme.DisabledColor())
		if i == 0 {
			tick.StrokeColor = theme.SuccessColor()
			tick.StrokeWidth = 3
		} else {
			tick.StrokeWidth = 2
		}

		tick.Position1 = fyne.NewPos(pivotX+sin*(needleLength-tickLength), pivotY-cos*(needleLength-tickLength))
		tick.Position2 = fyne.NewPos(pivotX+sin*needleLength, pivotY-cos*needleLength)
		meterObjects = append(meterObjects, tick)
	}

	needle.Position1 = fyne.NewPos(pivotX, pivotY)
	needle.Position2 = fyne.NewPos(pivotX, pivotY-needleLength)
	meterObjects = append(meterObjects, needle)

	analogMeter := container.NewCenter(container.New(&meterLayout{}, meterObjects...))

	var d dialog.Dialog
	listening := false
	var ticker *time.Ticker

	notes := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

	updateTuner := func(freq float64) {
		if freq < 20 {
			noteText.Text = "--"
			noteText.Color = theme.ForegroundColor()
			currentFreqLabel.SetText("Current: -- Hz")
			targetFreqLabel.SetText("Target: -- Hz")

			needle.Position2 = fyne.NewPos(pivotX, pivotY-needleLength)
			needle.StrokeColor = theme.DisabledColor()
			needle.Refresh()

			statusLabel.SetText("No signal")
			noteText.Refresh()
			return
		}

		halfSteps := 12.0 * math.Log2(freq/440.0)
		nearestStep := math.Round(halfSteps)
		cents := (halfSteps - nearestStep) * 100.0

		targetFreq := 440.0 * math.Pow(2.0, nearestStep/12.0)

		noteIndex := (int(nearestStep) + 9) % 12
		if noteIndex < 0 {
			noteIndex += 12
		}

		noteText.Text = notes[noteIndex]

		currentFreqLabel.SetText(fmt.Sprintf("Current: %.1f Hz", freq))
		targetFreqLabel.SetText(fmt.Sprintf("Target: %.1f Hz", targetFreq))

		displayCents := cents
		if displayCents < -50 {
			displayCents = -50
		}
		if displayCents > 50 {
			displayCents = 50
		}

		angle := (displayCents / 50.0) * (math.Pi / 3.0)
		sin := float32(math.Sin(angle))
		cos := float32(math.Cos(angle))

		needle.Position1 = fyne.NewPos(pivotX, pivotY)
		needle.Position2 = fyne.NewPos(pivotX+sin*needleLength, pivotY-cos*needleLength)

		if math.Abs(cents) < 5 {
			statusLabel.SetText("In Tune")
			noteText.Color = theme.SuccessColor()
			needle.StrokeColor = theme.SuccessColor()
		} else if cents < 0 {
			statusLabel.SetText("Too Flat")
			noteText.Color = theme.WarningColor()
			needle.StrokeColor = theme.WarningColor()
		} else {
			statusLabel.SetText("Too Sharp")
			noteText.Color = theme.WarningColor()
			needle.StrokeColor = theme.WarningColor()
		}

		noteText.Refresh()
		needle.Refresh()
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
					fluctuation := (rand.Float64() - 0.5) * 10.0
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

	hzContainer := container.NewHBox(layout.NewSpacer(), container.NewVBox(currentFreqLabel, targetFreqLabel), layout.NewSpacer())

	content := container.NewVBox(
		container.NewCenter(noteText),
		widget.NewLabel(""),
		hzContainer,
		widget.NewLabel(""),
		analogMeter,
		widget.NewLabel(""),
		statusLabel,
		widget.NewLabel(""),
		toggleBtn,
		widget.NewSeparator(),
		closeBtn,
	)

	d = dialog.NewCustomWithoutButtons("Instrument Tuner", container.NewPadded(content), w)
	d.Resize(fyne.NewSize(500, 650))
	d.Show()
}
