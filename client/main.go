package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/sysmock"
	"github.com/ziomciopoziomcio/digital-music-stand/client/ui"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Digital Music Stand")

	//todo: use real drivers
	netMgr := &sysmock.MockNetworkManager{Status: system.StatusDisconnected}
	pwrMgr := &sysmock.MockPowerManager{BatteryLevel: 85, Charging: true}
	medMgr := &sysmock.MockMediaManager{Volume: 50, Brightness: 80}
	devMgr := &sysmock.MockDeviceManager{IsAwake: true}

	mainWrapper := container.NewMax()

	var showDashboard func()
	var showSettings func()
	var showLogin func()

	showDashboard = func() {
		dash := ui.BuildDashboard(myWindow, myApp, showSettings, showLogin)
		mainWrapper.Objects = []fyne.CanvasObject{dash}
		mainWrapper.Refresh()
	}

	showSettings = func() {
		settingsView := ui.BuildSettings(myWindow, showDashboard, netMgr, pwrMgr, medMgr, devMgr)
		mainWrapper.Objects = []fyne.CanvasObject{settingsView}
		mainWrapper.Refresh()
	}

	showLogin = func() {
		loginView := ui.BuildLoginScreen(myApp, func(server, email, password string) error {
			token, err := network.Authenticate(server, email, password)
			if err == nil {
				myApp.Preferences().SetString("jwt_token", token)
				showDashboard()
			}
			return err
		}, showDashboard)
		mainWrapper.Objects = []fyne.CanvasObject{loginView}
		mainWrapper.Refresh()
	}

	showDashboard()

	myWindow.SetContent(mainWrapper)
	myWindow.Resize(fyne.NewSize(800, 480))
	myWindow.ShowAndRun()
}
