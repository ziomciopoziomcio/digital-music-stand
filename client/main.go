package main

import (
	"context"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/sysmock"
	"github.com/ziomciopoziomcio/digital-music-stand/client/ui"
	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver" // Import naszego nowego modułu!
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Digital Music Stand")

	dbMgr, err := localdb.NewDBManager("musicstand.db")
	if err != nil {
		panic(err)
	}

	wsMgr := webserver.NewManager("./scores")
	wsMgr.Start(8088)

	netMgr := &sysmock.MockNetworkManager{Status: system.StatusDisconnected}
	pwrMgr := &sysmock.MockPowerManager{BatteryLevel: 85, Charging: true}
	medMgr := &sysmock.MockMediaManager{Volume: 50, Brightness: 80}
	devMgr := &sysmock.MockDeviceManager{IsAwake: true}

	mainWrapper := container.NewMax()

	var showDashboard func()
	var showSettings func()
	var showLogin func()
	var showPractice func()
	var showConcert func()
	var showConcertSetup func()
	var showPairing func()

	showDashboard = func() {
		dash := ui.BuildDashboard(myWindow, myApp, showSettings, showLogin, showPractice, showConcert, showPairing)
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
				myApp.Preferences().SetString("server_addr", server)

				go func() {
					fmt.Println("Starting concerts sync...")
					conn, err := network.NewGRPCClient(server, token)
					if err != nil {
						fmt.Println("Error while connecting to gRPC:", err)
						return
					}
					defer conn.Close()

					concertClient := concertpb.NewConcertServiceClient(conn)
					err = network.SynchronizeConcerts(context.Background(), concertClient, dbMgr)
					if err != nil {
						fmt.Println("Concert sync error:", err)
					} else {
						fmt.Println("Concert synchronised successfully")
						//todo: refresh concert-related screens
					}
				}()

				showDashboard()
			}
			return err
		}, showDashboard)

		mainWrapper.Objects = []fyne.CanvasObject{loginView}
		mainWrapper.Refresh()
	}

	showPractice = func() {
		practiceView := ui.BuildPracticeMode(myWindow, dbMgr, showDashboard)
		mainWrapper.Objects = []fyne.CanvasObject{practiceView}
		mainWrapper.Refresh()
	}

	showConcert = func() {
		concertView := ui.BuildConcertMode(myWindow, dbMgr, showDashboard, showConcertSetup)
		mainWrapper.Objects = []fyne.CanvasObject{concertView}
		mainWrapper.Refresh()
	}

	showConcertSetup = func() {
		setupView := ui.BuildConcertSetup(myWindow, dbMgr, func(name, location, startTime string, setlist []localdb.Score) error {
			_, err := dbMgr.AddConcert(name, location, startTime, setlist)
			if err != nil {
				return err
			}

			token := myApp.Preferences().String("jwt_token")
			server := myApp.Preferences().String("server_addr")
			go func() {
				if err := network.CreateAndSyncConcert(server, token, name, location, startTime, setlist, dbMgr); err != nil {
					log.Printf("Background cloud sync error: %v", err)
				}
			}()

			return nil
		}, showConcert)
		mainWrapper.Objects = []fyne.CanvasObject{setupView}
		mainWrapper.Refresh()
	}

	showPairing = func() {
		ui.ShowPairingDialog(myWindow, wsMgr)
	}

	showDashboard()

	myWindow.SetContent(mainWrapper)
	myWindow.Resize(fyne.NewSize(800, 480))
	myWindow.ShowAndRun()
}
