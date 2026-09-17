package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/plugins"
)

func ShowPersonalMixerDialog(w fyne.Window) {
	mixer := plugins.GetActiveMixer()
	if mixer == nil || !mixer.GetConnectionStatus() {
		dialog.ShowInformation("Mixer Offline", "Connect to a stage mixer first in Settings.", w)
		return
	}

	busCount := mixer.GetBusCount()
	channelCount := mixer.GetChannelCount()

	if busCount == 0 || channelCount == 0 {
		dialog.ShowInformation("Unsupported", "This mixer plugin does not support personal buses.", w)
		return
	}

	currentBus := 1
	var d dialog.Dialog

	busLabel := widget.NewLabelWithStyle(fmt.Sprintf("Controlling: Bus %d", currentBus), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	var busOptions []string
	for i := 1; i <= busCount; i++ {
		busOptions = append(busOptions, fmt.Sprintf("Bus %d", i))
	}

	busSelect := widget.NewSelect(busOptions, func(selected string) {
		var b int
		fmt.Sscanf(selected, "Bus %d", &b)
		currentBus = b
		busLabel.SetText(fmt.Sprintf("Controlling: Bus %d", currentBus))
		// Opcjonalnie: pobieranie nowych wartości suwaków dla nowego busa
	})
	busSelect.SetSelected("Bus 1")

	channelsBox := container.NewHBox()
	for i := 1; i <= channelCount; i++ {
		chNum := i

		slider := widget.NewSlider(0, 1)
		slider.Orientation = widget.Vertical
		slider.SetValue(0.0)

		slider.OnChanged = func(val float64) {
			_ = mixer.SetChannelSendVolume(chNum, currentBus, val)
		}

		label := widget.NewLabelWithStyle(fmt.Sprintf("CH%02d", chNum), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		strip := container.NewBorder(label, nil, nil, nil, container.NewPadded(slider))
		channelsBox.Add(strip)
	}

	scrollableChannels := container.NewHScroll(channelsBox)

	masterBusSlider := widget.NewSlider(0, 1)
	masterBusSlider.Orientation = widget.Vertical
	masterBusSlider.SetValue(0.75)
	masterBusSlider.OnChanged = func(val float64) {
		_ = mixer.SetBusVolume(currentBus, val)
	}
	masterBusLabel := widget.NewLabelWithStyle("BUS\nMASTER", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	masterStrip := container.NewBorder(masterBusLabel, nil, nil, nil, container.NewPadded(masterBusSlider))

	mainMixerArea := container.NewBorder(nil, nil, nil, masterStrip, scrollableChannels)

	closeBtn := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		d.Hide()
	})

	header := container.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Personal Monitor Mixer - %s", mixer.Name()), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel("Select your IEM mix:"), busSelect),
		busLabel,
		widget.NewSeparator(),
	)

	content := container.NewBorder(header, container.NewPadded(closeBtn), nil, nil, mainMixerArea)

	d = dialog.NewCustomWithoutButtons("Personal Mixer", container.NewPadded(content), w)

	winSize := w.Canvas().Size()
	targetWidth := float32(800)
	targetHeight := float32(450)

	if winSize.Width < targetWidth {
		targetWidth = winSize.Width * 0.95
	}
	if winSize.Height < targetHeight {
		targetHeight = winSize.Height * 0.95
	}

	d.Resize(fyne.NewSize(targetWidth, targetHeight))
	d.Show()
}
