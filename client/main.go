package main

import (
	"context"
	"log"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/sysmock"
	"github.com/ziomciopoziomcio/digital-music-stand/client/ui"
	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
)

func main() {
	myApp := app.NewWithID("com.digitalmusicstand.client")
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
	var showConcertSetup func(editingConcert *localdb.Concert)
	var showPairing func()
	var showLockScreen func()
	var showInbox func()

	startBackgroundSync := func(server, token string) {
		go func() {
			log.Println("=== BACKGROUND SYNC STARTED ===")
			conn, err := network.NewGRPCClient(server, token)
			if err != nil {
				log.Printf("gRPC connection error: %v", err)
				return
			}
			defer conn.Close()

			scoreClient := scorepb.NewScoreServiceClient(conn)
			if err := network.SynchronizeScores(context.Background(), scoreClient, dbMgr); err != nil {
				log.Printf("SCORE SYNC ERROR: %v", err)
			} else {
				log.Println("SCORE SYNC SUCCESSFUL")
			}

			concertClient := concertpb.NewConcertServiceClient(conn)
			if err := network.SynchronizeConcerts(context.Background(), concertClient, dbMgr); err != nil {
				log.Printf("CONCERT SYNC ERROR: %v", err)
			} else {
				log.Println("CONCERT SYNC SUCCESSFUL")
			}

			bandClient := bandpb.NewBandServiceClient(conn)
			if err := network.SynchronizeInvitations(context.Background(), bandClient, dbMgr); err != nil {
				log.Printf("INVITATION SYNC ERROR: %v", err)
			} else {
				log.Println("INVITATION SYNC SUCCESSFUL")
			}
		}()
	}

	showDashboard = func() {
		dash := ui.BuildDashboard(myWindow, myApp, showSettings, showLogin, showPractice, showConcert, showPairing, showInbox)
		mainWrapper.Objects = []fyne.CanvasObject{dash}
		mainWrapper.Refresh()
	}

	showLockScreen = func() {
		lockView := ui.BuildLockScreen(myWindow, myApp, showDashboard)
		mainWrapper.Objects = []fyne.CanvasObject{lockView}
		mainWrapper.Refresh()
	}

	showSettings = func() {
		settingsView := ui.BuildSettings(myWindow, myApp, showDashboard, netMgr, pwrMgr, medMgr, devMgr)
		mainWrapper.Objects = []fyne.CanvasObject{settingsView}
		mainWrapper.Refresh()
	}

	showLogin = func() {
		loginView := ui.BuildLoginScreen(myApp, func(server, email, password string) error {
			token, err := network.Authenticate(server, email, password)
			if err == nil {
				myApp.Preferences().SetString("jwt_token", token)
				myApp.Preferences().SetString("server_addr", server)

				startBackgroundSync(server, token)
				showDashboard()
			}
			return err
		}, showDashboard)

		mainWrapper.Objects = []fyne.CanvasObject{loginView}
		mainWrapper.Refresh()
	}

	showPractice = func() {
		practiceView := ui.BuildPracticeMode(myWindow, dbMgr, func() {
			token := myApp.Preferences().String("jwt_token")
			server := myApp.Preferences().String("server_addr")
			if token != "" && server != "" {
				go func() {
					conn, err := network.NewGRPCClient(server, token)
					if err == nil {
						defer conn.Close()
						scoreClient := scorepb.NewScoreServiceClient(conn)
						if syncErr := network.SynchronizeScores(context.Background(), scoreClient, dbMgr); syncErr != nil {
							log.Printf("Background score sync error: %v", syncErr)
						}
					}
				}()
			}
		}, showDashboard)
		mainWrapper.Objects = []fyne.CanvasObject{practiceView}
		mainWrapper.Refresh()
	}

	triggerConcertSync := func() {
		token := myApp.Preferences().String("jwt_token")
		server := myApp.Preferences().String("server_addr")
		if token != "" && server != "" {
			go func() {
				conn, err := network.NewGRPCClient(server, token)
				if err == nil {
					defer conn.Close()
					concertClient := concertpb.NewConcertServiceClient(conn)
					if syncErr := network.SynchronizeConcerts(context.Background(), concertClient, dbMgr); syncErr != nil {
						log.Printf("Background concert sync error: %v", syncErr)
					}
				}
			}()
		}
	}

	showConcert = func() {
		concertView := ui.BuildConcertMode(myWindow, dbMgr, showDashboard, showConcertSetup, triggerConcertSync)
		mainWrapper.Objects = []fyne.CanvasObject{concertView}
		mainWrapper.Refresh()
	}

	showConcertSetup = func(editingConcert *localdb.Concert) {
		setupView := ui.BuildConcertSetup(myWindow, dbMgr, editingConcert, func(id, name, location, startTime string, setlist []localdb.SetlistItem) error {
			if id == "" {
				_, err := dbMgr.AddConcert(name, location, startTime, setlist)
				if err != nil {
					return err
				}
			} else {
				err := dbMgr.UpdateConcert(id, name, location, startTime, setlist)
				if err != nil {
					return err
				}
			}

			triggerConcertSync()
			return nil
		}, showConcert)
		mainWrapper.Objects = []fyne.CanvasObject{setupView}
		mainWrapper.Refresh()
	}

	showPairing = func() {
		ui.ShowPairingDialog(myWindow, wsMgr)
	}

	showInbox = func() {
		inboxView := ui.BuildInbox(myWindow, dbMgr, showDashboard, func(notif localdb.Notification, accept bool) {
			token := myApp.Preferences().String("jwt_token")
			server := myApp.Preferences().String("server_addr")

			if token != "" && server != "" {
				go func() {
					conn, err := network.NewGRPCClient(server, token)
					if err != nil {
						return
					}
					defer conn.Close()

					if notif.Type == "band_invite" {
						bandClient := bandpb.NewBandServiceClient(conn)
						invID, _ := strconv.ParseUint(notif.ReferenceID, 10, 32)

						_, err := bandClient.RespondToInvitation(context.Background(), &bandpb.RespondToInvitationRequest{
							InvitationId: uint32(invID),
							Accept:       accept,
						})
						if err != nil {
							log.Printf("Failed to respond to invitation: %v", err)
						} else {
							log.Printf("Successfully responded to invitation %d (Accepted: %v)", invID, accept)
						}
					}
				}()
			}
		})
		mainWrapper.Objects = []fyne.CanvasObject{inboxView}
		mainWrapper.Refresh()
	}

	savedToken := myApp.Preferences().String("jwt_token")
	savedServer := myApp.Preferences().String("server_addr")

	if savedToken != "" && savedServer != "" {
		log.Println("Saved login data detected. Auto-login and background sync initiated...")
		startBackgroundSync(savedServer, savedToken)
	}

	savedPin := myApp.Preferences().String("app_pin")
	if savedPin != "" {
		showLockScreen()
	} else {
		showDashboard()
	}

	myWindow.SetContent(mainWrapper)
	myWindow.Resize(fyne.NewSize(800, 480))
	myWindow.ShowAndRun()
}
