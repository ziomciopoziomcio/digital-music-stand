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

func ShowPersonalMixerDialog(w fyne.Window, a fyne.App, profileID string) {
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

	prefBus := profileID + "_personal_bus"
	currentBus := a.Preferences().IntWithFallback(prefBus, 1)
	if currentBus < 1 || currentBus > busCount {
		currentBus = 1
	}

	var d dialog.Dialog
	busLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	var busOptions []string
	busMap := make(map[string]int)

	for i := 1; i <= busCount; i++ {
		name, err := mixer.GetBusName(i)
		if err != nil || name == "" {
			name = fmt.Sprintf("Bus %d", i)
		}
		opt := fmt.Sprintf("%d: %s", i, name)
		busOptions = append(busOptions, opt)
		busMap[opt] = i
	}

	var updateSliders func()

	busSelect := widget.NewSelect(busOptions, func(selected string) {
		if b, ok := busMap[selected]; ok {
			currentBus = b
			a.Preferences().SetInt(prefBus, currentBus)
			busLabel.SetText(fmt.Sprintf("Controlling: %s", selected))
			if updateSliders != nil {
				updateSliders()
			}
		}
	})

	var initialSelect string
	for k, v := range busMap {
		if v == currentBus {
			initialSelect = k
			break
		}
	}
	busSelect.SetSelected(initialSelect)
	busLabel.SetText(fmt.Sprintf("Controlling: %s", initialSelect))

	channelsBox := container.NewHBox()
	var sliders []*widget.Slider

	for i := 1; i <= channelCount; i++ {
		slider := widget.NewSlider(0, 1)
		slider.Orientation = widget.Vertical
		slider.Step = 0.01
		sliders = append(sliders, slider)

		name, err := mixer.GetChannelName(i)
		if err != nil || name == "" {
			name = fmt.Sprintf("CH%02d", i)
		}
		label := widget.NewLabelWithStyle(name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		strip := container.NewBorder(label, nil, nil, nil, container.NewPadded(slider))
		channelsBox.Add(strip)
	}

	scrollableChannels := container.NewHScroll(channelsBox)

	masterBusSlider := widget.NewSlider(0, 1)
	masterBusSlider.Orientation = widget.Vertical
	masterBusSlider.Step = 0.01

	masterBusLabel := widget.NewLabelWithStyle("MASTER", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	masterStrip := container.NewBorder(masterBusLabel, nil, nil, nil, container.NewPadded(masterBusSlider))

	isUpdating := false

	updateSliders = func() {
		isUpdating = true
		defer func() { isUpdating = false }()

		for i, sl := range sliders {
			chNum := i + 1
			vol, _ := mixer.GetChannelSendVolume(chNum, currentBus)
			sl.SetValue(vol)
		}
		mVol, _ := mixer.GetBusVolume(currentBus)
		masterBusSlider.SetValue(mVol)
	}

	for i, sl := range sliders {
		chNum := i + 1
		slider := sl
		slider.OnChanged = func(val float64) {
			if isUpdating {
				return
			}
			_ = mixer.SetChannelSendVolume(chNum, currentBus, val)
		}
	}

	masterBusSlider.OnChanged = func(val float64) {
		if isUpdating {
			return
		}
		_ = mixer.SetBusVolume(currentBus, val)
	}

	updateSliders()

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

	d.Resize(w.Canvas().Size())
	d.Show()
}
