package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/ziomciopoziomcio/digital-music-stand/client/ui"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Digital Music Stand")

	dashboard := ui.BuildDashboard(myWindow)

	myWindow.SetContent(dashboard)
	myWindow.Resize(fyne.NewSize(800, 480))
	myWindow.ShowAndRun()
}
