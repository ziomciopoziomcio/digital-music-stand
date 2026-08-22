package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/ui"
	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
)

var AppVersion = "client-v0.1.0-alpha.1"

func main() {
	myApp := app.NewWithID("com.digitalmusicstand.client")
	myWindow := myApp.NewWindow("Digital Music Stand")

	dbMgr, err := localdb.NewDBManager("musicstand.db")
	if err != nil {
		panic(err)
	}

	wsMgr := webserver.NewManager("./scores")
	wsMgr.Start(8088)

	netMgr, pwrMgr, medMgr, devMgr := InitManagers()

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
	var showProfile func()

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
				log.Printf("BAND INVITATION SYNC ERROR: %v", err)
			} else {
				log.Println("BAND INVITATION SYNC SUCCESSFUL")
			}

			if err := network.SynchronizeConcertInvitations(context.Background(), concertClient, dbMgr); err != nil {
				log.Printf("CONCERT INVITATION SYNC ERROR: %v", err)
			} else {
				log.Println("CONCERT INVITATION SYNC SUCCESSFUL")
			}

			if err := network.SynchronizeScoreInvitations(context.Background(), scoreClient, dbMgr); err != nil {
				log.Printf("SCORE INVITATION SYNC ERROR: %v", err)
			} else {
				log.Println("SCORE INVITATION SYNC SUCCESSFUL")
			}
		}()
	}

	showDashboard = func() {
		dash := ui.BuildDashboard(myWindow, myApp, showSettings, showLogin, showPractice, showConcert, showPairing, showInbox, showProfile)
		mainWrapper.Objects = []fyne.CanvasObject{dash}
		mainWrapper.Refresh()
	}

	showLockScreen = func() {
		if ui.SetQuickSettingsVisible != nil {
			ui.SetQuickSettingsVisible(false)
		}
		lockView := ui.BuildLockScreen(myWindow, myApp, func() {
			if ui.SetQuickSettingsVisible != nil {
				ui.SetQuickSettingsVisible(true)
			}
			showDashboard()
		})
		mainWrapper.Objects = []fyne.CanvasObject{lockView}
		mainWrapper.Refresh()
	}

	showSettings = func() {
		settingsView := ui.BuildSettings(myWindow, myApp, AppVersion, showDashboard, netMgr, pwrMgr, medMgr, devMgr)
		mainWrapper.Objects = []fyne.CanvasObject{settingsView}
		mainWrapper.Refresh()
	}

	showLogin = func() {
		loginView := ui.BuildLoginScreen(
			myWindow,
			myApp,
			func(server, email, password string) error {
				token, err := network.Authenticate(server, email, password)
				if err == nil {
					myApp.Preferences().SetString("jwt_token", token)
					myApp.Preferences().SetString("server_addr", server)

					myApp.Preferences().SetString("user_email", email)
					myApp.Preferences().SetString("user_password", password)

					startBackgroundSync(server, token)
					showDashboard()
				}
				return err
			},
			func(server, email, password, name, surname string) (string, error) {
				conn, err := network.NewGRPCClient(server, "")
				if err != nil {
					return "", fmt.Errorf("connection failed: %w", err)
				}
				defer conn.Close()

				client := userpb.NewUserServiceClient(conn)
				resp, err := client.RegisterUser(context.Background(), &userpb.RegisterUserRequest{
					Email:    email,
					Password: password,
					Name:     name,
					Surname:  surname,
				})
				if err != nil {
					return "", err
				}
				return resp.GetMessage(), nil
			},
			func(server, email string) (string, error) {
				conn, err := network.NewGRPCClient(server, "")
				if err != nil {
					return "", fmt.Errorf("connection failed: %w", err)
				}
				defer conn.Close()

				client := userpb.NewUserServiceClient(conn)
				resp, err := client.ResetPassword(context.Background(), &userpb.ResetPasswordRequest{
					Email: email,
				})
				if err != nil {
					return "", err
				}
				return resp.GetMessage(), nil
			},
			showDashboard,
		)

		mainWrapper.Objects = []fyne.CanvasObject{loginView}
		mainWrapper.Refresh()
	}

	showPractice = func() {
		practiceView := ui.BuildPracticeMode(myWindow, myApp, dbMgr, func() {
			token := myApp.Preferences().String("jwt_token")
			server := myApp.Preferences().String("server_addr")
			if token != "" && server != "" {
				go func() {
					conn, err := network.NewGRPCClient(server, token)
					if err == nil {
						defer conn.Close()
						scoreClient := scorepb.NewScoreServiceClient(conn)
						_ = network.SynchronizeScores(context.Background(), scoreClient, dbMgr)
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
		concertView := ui.BuildConcertMode(myWindow, myApp, dbMgr, showDashboard, showConcertSetup, triggerConcertSync)
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

					invID, _ := strconv.ParseUint(notif.ReferenceID, 10, 32)

					switch notif.Type {
					case "band_invite":
						bandClient := bandpb.NewBandServiceClient(conn)
						_, _ = bandClient.RespondToInvitation(context.Background(), &bandpb.RespondToInvitationRequest{
							InvitationId: uint32(invID),
							Accept:       accept,
						})

					case "concert_invite":
						concertClient := concertpb.NewConcertServiceClient(conn)
						_, _ = concertClient.RespondToConcertInvitation(context.Background(), &concertpb.RespondToConcertInvitationRequest{
							InvitationId: uint32(invID),
							Accept:       accept,
						})

					case "score_invite":
						scoreClient := scorepb.NewScoreServiceClient(conn)
						_, _ = scoreClient.RespondToScoreInvitation(context.Background(), &scorepb.RespondToScoreInvitationRequest{
							InvitationId: uint32(invID),
							Accept:       accept,
						})
					}

					if accept {
						startBackgroundSync(server, token)
					}
				}()
			}
		})
		mainWrapper.Objects = []fyne.CanvasObject{inboxView}
		mainWrapper.Refresh()
	}

	showProfile = func() {
		profileView := ui.BuildProfile(
			myWindow,
			myApp,
			showDashboard,
			func() {
				myApp.Preferences().SetString("jwt_token", "")
				myApp.Preferences().SetString("server_addr", "")
				myApp.Preferences().SetString("user_email", "")
				myApp.Preferences().SetString("user_password", "")
				showDashboard()
			},
			func() ([]ui.BandInfo, error) {
				token := myApp.Preferences().String("jwt_token")
				server := myApp.Preferences().String("server_addr")
				if token == "" || server == "" {
					return nil, fmt.Errorf("not logged in")
				}

				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return nil, err
				}
				defer conn.Close()

				bandClient := bandpb.NewBandServiceClient(conn)
				resp, err := bandClient.ListMyBands(context.Background(), &bandpb.ListMyBandsRequest{})
				if err != nil {
					return nil, err
				}

				var bands []ui.BandInfo
				for _, b := range resp.GetBands() {
					bands = append(bands, ui.BandInfo{
						ID:        b.Id,
						Name:      b.Name,
						IsManager: b.IsManager,
					})
				}
				return bands, nil
			},
			func(name string) error {
				token := myApp.Preferences().String("jwt_token")
				server := myApp.Preferences().String("server_addr")
				if token == "" || server == "" {
					return fmt.Errorf("not logged in")
				}

				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return err
				}
				defer conn.Close()

				bandClient := bandpb.NewBandServiceClient(conn)
				_, err = bandClient.CreateBand(context.Background(), &bandpb.CreateBandRequest{Name: name})
				return err
			},
			func(bandID uint32, email string) error {
				token := myApp.Preferences().String("jwt_token")
				server := myApp.Preferences().String("server_addr")
				if token == "" || server == "" {
					return fmt.Errorf("not logged in")
				}

				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return err
				}
				defer conn.Close()

				bandClient := bandpb.NewBandServiceClient(conn)
				_, err = bandClient.InviteMember(context.Background(), &bandpb.InviteMemberRequest{
					BandId:       bandID,
					InviteeEmail: email,
				})
				return err
			},
			func(oldPassword, newPassword string) error {
				token := myApp.Preferences().String("jwt_token")
				server := myApp.Preferences().String("server_addr")
				if token == "" || server == "" {
					return fmt.Errorf("not logged in")
				}

				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return err
				}
				defer conn.Close()

				userClient := userpb.NewUserServiceClient(conn)
				_, err = userClient.ChangePassword(context.Background(), &userpb.ChangePasswordRequest{
					OldPassword: oldPassword,
					NewPassword: newPassword,
				})
				return err
			},
			func(bandID uint32) ([]ui.MemberInfo, error) {
				token := myApp.Preferences().String("jwt_token")
				server := myApp.Preferences().String("server_addr")
				if token == "" || server == "" {
					return nil, fmt.Errorf("not logged in")
				}

				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return nil, err
				}
				defer conn.Close()

				bandClient := bandpb.NewBandServiceClient(conn)
				resp, err := bandClient.ListBandMembers(context.Background(), &bandpb.ListBandMembersRequest{BandId: bandID})
				if err != nil {
					return nil, err
				}

				var members []ui.MemberInfo
				for _, m := range resp.GetMembers() {
					members = append(members, ui.MemberInfo{
						UserID:  m.GetUserId(),
						Email:   m.GetEmail(),
						Name:    m.GetName(),
						Surname: m.GetSurname(),
						Role:    m.GetRole(),
					})
				}
				return members, nil
			},
			func(bandID uint32, userID uint32, email string) error {
				token := myApp.Preferences().String("jwt_token")
				server := myApp.Preferences().String("server_addr")
				if token == "" || server == "" {
					return fmt.Errorf("not logged in")
				}

				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return err
				}
				defer conn.Close()

				bandClient := bandpb.NewBandServiceClient(conn)
				_, err = bandClient.RemoveMember(context.Background(), &bandpb.RemoveMemberRequest{
					BandId: bandID,
					UserId: userID,
					Email:  email,
				})
				return err
			},
		)
		mainWrapper.Objects = []fyne.CanvasObject{profileView}
		mainWrapper.Refresh()
	}

	savedToken := myApp.Preferences().String("jwt_token")
	savedServer := myApp.Preferences().String("server_addr")

	if savedToken != "" && savedServer != "" {
		log.Println("Saved login data detected. Initiating background sync...")

		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				serverAddr := myApp.Preferences().String("server_addr")
				token := myApp.Preferences().String("jwt_token")

				if serverAddr == "" || token == "" {
					<-ticker.C
					continue
				}

				conn, err := network.NewGRPCClient(serverAddr, token)
				if err != nil {
					log.Printf("[Sync] Server unreachable: %v", err)
					<-ticker.C
					continue
				}

				userClient := userpb.NewUserServiceClient(conn)
				ctx := context.Background()

				_, profileErr := userClient.GetProfile(ctx, &userpb.GetProfileRequest{})
				if profileErr != nil {
					conn.Close()
					log.Printf("[Sync] Authorization failed (token expired?): %v", profileErr)

					email := myApp.Preferences().String("user_email")
					password := myApp.Preferences().String("user_password")

					if email != "" && password != "" {
						log.Println("[Sync] Attempting automatic re-login...")
						newToken, authErr := network.Authenticate(serverAddr, email, password)

						if authErr == nil {
							log.Println("[Sync] Automatic re-login successful! Updating token.")
							myApp.Preferences().SetString("jwt_token", newToken)
							continue
						} else {
							log.Printf("[Sync] Auto re-login failed: %v", authErr)
						}
					}

					myApp.Preferences().SetString("jwt_token", "")
					log.Println("[Sync] Background sync stopped. Running in pure offline mode.")
					return
				}

				scoreClient := scorepb.NewScoreServiceClient(conn)
				concertClient := concertpb.NewConcertServiceClient(conn)

				syncScoreErr := network.SynchronizeScores(ctx, scoreClient, dbMgr)
				syncConcertErr := network.SynchronizeConcerts(ctx, concertClient, dbMgr)
				conn.Close()

				if syncScoreErr != nil || syncConcertErr != nil {
					log.Printf("[Sync] Sync errors - Scores: %v, Concerts: %v", syncScoreErr, syncConcertErr)
				} else {
					log.Println("[Sync] Synchronization successful.")
				}

				<-ticker.C
			}
		}()
	}

	appWithQuickSettings := ui.WrapWithQuickSettings(myWindow, myApp, mainWrapper)

	savedPin := myApp.Preferences().String("app_pin")
	if savedPin != "" {
		showLockScreen()
	} else {
		showDashboard()
	}

	myWindow.SetContent(appWithQuickSettings)
	myWindow.Resize(fyne.NewSize(800, 480))
	myWindow.ShowAndRun()
}
