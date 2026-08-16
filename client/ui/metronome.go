package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
)

func ShowMetronomeDialog(w fyne.Window) {
	bpm := 120
	beatsPerMeasure := 4
	playing := false

	var d dialog.Dialog

	metroAudio, _ := audio.NewMetronomeAudio()
	if metroAudio != nil {
		metroAudio.SetBPM(float64(bpm))
		metroAudio.SetTimeSignature(beatsPerMeasure)
	}

	bpmLabel := widget.NewLabelWithStyle(fmt.Sprintf("%d BPM", bpm), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	beatLabel := widget.NewLabelWithStyle("4/4 Time", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	indicator := canvas.NewRectangle(theme.DisabledColor())
	indicator.SetMinSize(fyne.NewSize(150, 150))
	indicatorContainer := container.NewCenter(indicator)

	bpmSlider := widget.NewSlider(40, 240)
	bpmSlider.SetValue(float64(bpm))

	updateBPM := func(newBPM int) {
		if newBPM < 40 {
			newBPM = 40
		}
		if newBPM > 240 {
			newBPM = 240
		}
		bpm = newBPM
		bpmSlider.SetValue(float64(bpm))
		bpmLabel.SetText(fmt.Sprintf("%d BPM", bpm))

		if metroAudio != nil {
			metroAudio.SetBPM(float64(bpm))
		}
	}

	bpmSlider.OnChanged = func(val float64) {
		updateBPM(int(val))
	}

	minusBtn := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() { updateBPM(bpm - 1) })
	plusBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() { updateBPM(bpm + 1) })

	bpmControls := container.NewBorder(nil, nil, minusBtn, plusBtn, bpmSlider)

	measureBtn := widget.NewButton("Time Sig: 4/4", nil)
	measureBtn.OnTapped = func() {
		beatsPerMeasure++
		if beatsPerMeasure > 7 {
			beatsPerMeasure = 1
		}
		if beatsPerMeasure == 1 {
			measureBtn.SetText("Time Sig: Off")
			beatLabel.SetText("Free mode")
		} else {
			measureBtn.SetText(fmt.Sprintf("Time Sig: %d/4", beatsPerMeasure))
			beatLabel.SetText(fmt.Sprintf("%d/4 Time", beatsPerMeasure))
		}

		if metroAudio != nil {
			metroAudio.SetTimeSignature(beatsPerMeasure)
		}
	}

	toggleBtn := widget.NewButtonWithIcon("Start Metronome", theme.MediaPlayIcon(), nil)
	toggleBtn.Importance = widget.HighImportance

	if metroAudio != nil {
		metroAudio.OnBeat = func(isAccent bool) {
			if isAccent {
				indicator.FillColor = theme.SuccessColor()
			} else {
				indicator.FillColor = theme.PrimaryColor()
			}
			indicator.Refresh()

			time.AfterFunc(100*time.Millisecond, func() {
				indicator.FillColor = theme.DisabledColor()
				indicator.Refresh()
			})
		}
	}

	toggleBtn.OnTapped = func() {
		playing = !playing
		if playing {
			toggleBtn.SetText("Stop Metronome")
			toggleBtn.SetIcon(theme.MediaStopIcon())

			if metroAudio != nil {
				metroAudio.Start()
			}
		} else {
			toggleBtn.SetText("Start Metronome")
			toggleBtn.SetIcon(theme.MediaPlayIcon())

			if metroAudio != nil {
				metroAudio.Stop()
			}
		}
	}

	closeBtn := widget.NewButton("Close", func() {
		if metroAudio != nil {
			metroAudio.Stop()
			metroAudio.Close()
		}
		d.Hide()
	})

	content := container.NewVBox(
		indicatorContainer,
		widget.NewLabel(""),
		bpmLabel,
		beatLabel,
		bpmControls,
		measureBtn,
		widget.NewLabel(""),
		toggleBtn,
		widget.NewSeparator(),
		closeBtn,
	)

	d = dialog.NewCustomWithoutButtons("Metronome", container.NewPadded(content), w)
	d.Resize(fyne.NewSize(400, 550))
	d.Show()
}
