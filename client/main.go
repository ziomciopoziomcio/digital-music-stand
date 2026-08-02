package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"github.com/ziomciopoziomcio/digital-music-stand/client/ui"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Digital Music Stand")

	mainWrapper := container.NewMax()

	var showDashboard func()
	var showSettings func()

	showDashboard = func() {
		dash := ui.BuildDashboard(myWindow, showSettings)
		mainWrapper.Objects = []fyne.CanvasObject{dash}
		mainWrapper.Refresh()
	}

	showSettings = func() {
		settingsView := ui.BuildSettings(myWindow, showDashboard)
		mainWrapper.Objects = []fyne.CanvasObject{settingsView}
		mainWrapper.Refresh()
	}

	showDashboard()

	myWindow.SetContent(mainWrapper)
	myWindow.Resize(fyne.NewSize(800, 480))
	myWindow.ShowAndRun()
}
