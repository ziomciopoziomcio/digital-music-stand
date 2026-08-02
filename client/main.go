package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/sysmock"
	"github.com/ziomciopoziomcio/digital-music-stand/client/ui"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Digital Music Stand")

	//todo: use real drivers
	netMgr := &sysmock.MockNetworkManager{Status: system.StatusDisconnected}
	pwrMgr := &sysmock.MockPowerManager{BatteryLevel: 85, Charging: false}
	medMgr := &sysmock.MockMediaManager{Volume: 50, Brightness: 80}
	devMgr := &sysmock.MockDeviceManager{IsAwake: true}

	mainWrapper := container.NewMax()

	var showDashboard func()
	var showSettings func()

	showDashboard = func() {
		dash := ui.BuildDashboard(myWindow, showSettings)
		mainWrapper.Objects = []fyne.CanvasObject{dash}
		mainWrapper.Refresh()
	}

	showSettings = func() {
		settingsView := ui.BuildSettings(myWindow, showDashboard, netMgr, pwrMgr, medMgr, devMgr)
		mainWrapper.Objects = []fyne.CanvasObject{settingsView}
		mainWrapper.Refresh()
	}

	showDashboard()

	myWindow.SetContent(mainWrapper)
	myWindow.Resize(fyne.NewSize(800, 480))
	myWindow.ShowAndRun()
}
