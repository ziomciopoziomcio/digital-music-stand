package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/profiles"
	"github.com/ziomciopoziomcio/digital-music-stand/client/ui"
	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
)

var AppVersion = "client-v0.1.0-alpha.1"

func main() {
	myApp := app.NewWithID("com.digitalmusicstand.client")
	myWindow := myApp.NewWindow("Digital Music Stand")
	ui.ApplyAppTheme(myApp, "blue")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}
	appDataDir := filepath.Join(homeDir, ".digitalmusicstand")

	pm, err := profiles.NewManager(appDataDir)
	if err != nil {
		log.Fatalf("Fatal: %v", err)
	}

	ui.ShowProfileSelector(myWindow, myApp, pm, func(profileID string) {
		launchProfileSession(myWindow, myApp, pm, profileID)
	})

	myWindow.Resize(fyne.NewSize(800, 480))
	myWindow.ShowAndRun()
}

func launchProfileSession(myWindow fyne.Window, myApp fyne.App, pm *profiles.Manager, profileID string) {
	sessionCtx, cancelSession := context.WithCancel(context.Background())

	prefToken := profileID + "_jwt_token"
	prefRefresh := profileID + "_refresh_token"
	prefServer := profileID + "_server_addr"
	prefTheme := profileID + "_theme_color"

	themeColor := myApp.Preferences().StringWithFallback(prefTheme, "blue")
	ui.ApplyAppTheme(myApp, themeColor)

	profilePath := pm.GetProfilePath(profileID)
	dbPath := filepath.Join(profilePath, "musicstand.db")
	scoresPath := filepath.Join(profilePath, "scores")

	dbMgr, err := localdb.NewDBManager(dbPath)
	if err != nil {
		panic(err)
	}

	wsMgr := webserver.NewManager(scoresPath)
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
	var showInbox func()
	var showProfile func()
	var showLockScreen func()

	isCloudConnected := func() bool {
		token := myApp.Preferences().String(prefToken)
		server := myApp.Preferences().String(prefServer)
		if len(token) > 50 && server != "" {
			return true
		}
		return false
	}

	onSwitchProfile := func() {
		cancelSession()
		ui.ApplyAppTheme(myApp, "blue")
		ui.ShowProfileSelector(myWindow, myApp, pm, func(newProfileID string) {
			launchProfileSession(myWindow, myApp, pm, newProfileID)
		})
	}

	startBackgroundSyncLoop := func(server, initialToken string) {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			doSync := func() bool {
				currentToken := myApp.Preferences().String(prefToken)
				if currentToken == "" {
					return false
				}

				conn, err := network.NewGRPCClient(server, currentToken)
				if err != nil {
					return true
				}
				defer conn.Close()

				userClient := userpb.NewUserServiceClient(conn)
				_, profileErr := userClient.GetProfile(sessionCtx, &userpb.GetProfileRequest{})

				if profileErr != nil {
					st, ok := status.FromError(profileErr)
					if ok && (st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded) {
						return true
					}

					if ok && st.Code() == codes.Unauthenticated {
						refreshToken := myApp.Preferences().String(prefRefresh)
						if refreshToken != "" {
							newJwt, newRef, err := network.RefreshSession(server, refreshToken)
							if err == nil {
								myApp.Preferences().SetString(prefToken, newJwt)
								myApp.Preferences().SetString(prefRefresh, newRef)
								return true
							}
						}
					}

					myApp.Preferences().SetString(prefToken, "")
					myApp.Preferences().SetString(prefRefresh, "")
					return false
				}

				scoreClient := scorepb.NewScoreServiceClient(conn)
				concertClient := concertpb.NewConcertServiceClient(conn)
				bandClient := bandpb.NewBandServiceClient(conn)

				if err := network.SynchronizeScores(sessionCtx, scoreClient, dbMgr, scoresPath); err != nil {
					log.Printf("Score sync error: %v", err)
				}
				if err := network.SynchronizeConcerts(sessionCtx, concertClient, dbMgr); err != nil {
					log.Printf("Concert sync error: %v", err)
				}
				if err := network.SynchronizeInvitations(sessionCtx, bandClient, dbMgr); err != nil {
					log.Printf("Band invite sync error: %v", err)
				}
				if err := network.SynchronizeConcertInvitations(sessionCtx, concertClient, dbMgr); err != nil {
					log.Printf("Concert invite sync error: %v", err)
				}
				if err := network.SynchronizeScoreInvitations(sessionCtx, scoreClient, dbMgr); err != nil {
					log.Printf("Score invite sync error: %v", err)
				}

				return true
			}

			if !doSync() {
				return
			}

			for {
				select {
				case <-sessionCtx.Done():
					return
				case <-ticker.C:
					if !doSync() {
						return
					}
				}
			}
		}()
	}

	showDashboard = func() {
		dash := ui.BuildDashboard(myWindow, myApp, showSettings, showLogin, showPractice, showConcert, showPairing, showInbox, showProfile, isCloudConnected)
		mainWrapper.Objects = []fyne.CanvasObject{dash}
		mainWrapper.Refresh()
	}

	showLockScreen = func() {
		profileInfo, err := pm.GetProfiles()
		hasPin := false
		if err == nil {
			for _, p := range profileInfo {
				if p.ID == profileID && p.PinHash != "" {
					hasPin = true
					break
				}
			}
		}

		if !hasPin {
			dialog.ShowInformation("No PIN", "Profile has no PIN set.", myWindow)
			return
		}

		var previousView fyne.CanvasObject
		if len(mainWrapper.Objects) > 0 {
			previousView = mainWrapper.Objects[0]
		}

		if ui.SetQuickSettingsVisible != nil {
			ui.SetQuickSettingsVisible(false)
		}

		lockView := ui.BuildLockScreen(myWindow, myApp,
			func(enteredPin string) bool {
				return pm.VerifyPin(profileID, enteredPin)
			},
			func() {
				if ui.SetQuickSettingsVisible != nil {
					ui.SetQuickSettingsVisible(true)
				}
				if previousView != nil {
					mainWrapper.Objects = []fyne.CanvasObject{previousView}
					mainWrapper.Refresh()
				} else {
					showDashboard()
				}
			},
		)

		mainWrapper.Objects = []fyne.CanvasObject{lockView}
		mainWrapper.Refresh()
	}

	showSettings = func() {
		settingsView := ui.BuildSettings(myWindow, myApp, AppVersion, showDashboard, netMgr, pwrMgr, medMgr, devMgr, pm, profileID)
		mainWrapper.Objects = []fyne.CanvasObject{settingsView}
		mainWrapper.Refresh()
	}

	showLogin = func() {
		loginView := ui.BuildLoginScreen(
			myWindow,
			myApp,
			func(server, email, password string) error {
				token, refreshToken, err := network.Authenticate(server, email, password)
				if err == nil {
					myApp.Preferences().SetString(prefToken, token)
					myApp.Preferences().SetString(prefRefresh, refreshToken)
					myApp.Preferences().SetString(prefServer, server)
					startBackgroundSyncLoop(server, token)
					showDashboard()
				}
				return err
			},
			func(server, email, password, name, surname string) (string, error) {
				conn, err := network.NewGRPCClient(server, "")
				if err != nil {
					return "", err
				}
				defer conn.Close()

				client := userpb.NewUserServiceClient(conn)
				resp, err := client.RegisterUser(sessionCtx, &userpb.RegisterUserRequest{
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
					return "", err
				}
				defer conn.Close()

				client := userpb.NewUserServiceClient(conn)
				resp, err := client.ResetPassword(sessionCtx, &userpb.ResetPasswordRequest{
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
			token := myApp.Preferences().String(prefToken)
			server := myApp.Preferences().String(prefServer)
			if token != "" && server != "" {
				go func() {
					conn, err := network.NewGRPCClient(server, token)
					if err == nil {
						defer conn.Close()
						scoreClient := scorepb.NewScoreServiceClient(conn)
						_ = network.SynchronizeScores(sessionCtx, scoreClient, dbMgr, scoresPath)
					}
				}()
			}
		}, showDashboard)
		mainWrapper.Objects = []fyne.CanvasObject{practiceView}
		mainWrapper.Refresh()
	}

	triggerConcertSync := func() {
		token := myApp.Preferences().String(prefToken)
		server := myApp.Preferences().String(prefServer)
		if token != "" && server != "" {
			go func() {
				conn, err := network.NewGRPCClient(server, token)
				if err == nil {
					defer conn.Close()
					concertClient := concertpb.NewConcertServiceClient(conn)
					_ = network.SynchronizeConcerts(sessionCtx, concertClient, dbMgr)
				}
			}()
		}
	}

	showConcert = func() {
		refreshToken := myApp.Preferences().String(prefRefresh)

		if refreshToken != "" {
			token, _, err := new(jwt.Parser).ParseUnverified(refreshToken, jwt.MapClaims{})
			if err == nil {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if exp, ok := claims["exp"].(float64); ok {
						if time.Until(time.Unix(int64(exp), 0)) < 24*time.Hour {
							dialog.ShowConfirm("Warning",
								"Session expiring soon.",
								func(confirm bool) {
									if confirm {
										concertView := ui.BuildConcertMode(myWindow, myApp, dbMgr, showDashboard, showConcertSetup, triggerConcertSync)
										mainWrapper.Objects = []fyne.CanvasObject{concertView}
										mainWrapper.Refresh()
									}
								}, myWindow)
							return
						}
					}
				}
			}
		}

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
			token := myApp.Preferences().String(prefToken)
			server := myApp.Preferences().String(prefServer)

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
						_, _ = bandClient.RespondToInvitation(sessionCtx, &bandpb.RespondToInvitationRequest{
							InvitationId: uint32(invID),
							Accept:       accept,
						})
					case "concert_invite":
						concertClient := concertpb.NewConcertServiceClient(conn)
						_, _ = concertClient.RespondToConcertInvitation(sessionCtx, &concertpb.RespondToConcertInvitationRequest{
							InvitationId: uint32(invID),
							Accept:       accept,
						})
					case "score_invite":
						scoreClient := scorepb.NewScoreServiceClient(conn)
						_, _ = scoreClient.RespondToScoreInvitation(sessionCtx, &scorepb.RespondToScoreInvitationRequest{
							InvitationId: uint32(invID),
							Accept:       accept,
						})
					}

					if accept {
						triggerConcertSync()
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
				myApp.Preferences().SetString(prefToken, "")
				myApp.Preferences().SetString(prefRefresh, "")
				myApp.Preferences().SetString(prefServer, "")
				cancelSession()
				ui.ApplyAppTheme(myApp, "blue")
				ui.ShowProfileSelector(myWindow, myApp, pm, func(profileID string) {
					launchProfileSession(myWindow, myApp, pm, profileID)
				})
			},
			func() ([]ui.BandInfo, error) {
				token := myApp.Preferences().String(prefToken)
				server := myApp.Preferences().String(prefServer)
				if token == "" || server == "" {
					return nil, fmt.Errorf("not logged in")
				}
				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return nil, err
				}
				defer conn.Close()
				bandClient := bandpb.NewBandServiceClient(conn)
				resp, err := bandClient.ListMyBands(sessionCtx, &bandpb.ListMyBandsRequest{})
				if err != nil {
					return nil, err
				}
				var bands []ui.BandInfo
				for _, b := range resp.GetBands() {
					bands = append(bands, ui.BandInfo{ID: b.Id, Name: b.Name, IsManager: b.IsManager})
				}
				return bands, nil
			},
			func(name string) error {
				token := myApp.Preferences().String(prefToken)
				server := myApp.Preferences().String(prefServer)
				if token == "" || server == "" {
					return fmt.Errorf("not logged in")
				}
				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return err
				}
				defer conn.Close()
				bandClient := bandpb.NewBandServiceClient(conn)
				_, err = bandClient.CreateBand(sessionCtx, &bandpb.CreateBandRequest{Name: name})
				return err
			},
			func(bandID uint32, email string) error {
				token := myApp.Preferences().String(prefToken)
				server := myApp.Preferences().String(prefServer)
				if token == "" || server == "" {
					return fmt.Errorf("not logged in")
				}
				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return err
				}
				defer conn.Close()
				bandClient := bandpb.NewBandServiceClient(conn)
				_, err = bandClient.InviteMember(sessionCtx, &bandpb.InviteMemberRequest{BandId: bandID, InviteeEmail: email})
				return err
			},
			func(oldPassword, newPassword string) error {
				token := myApp.Preferences().String(prefToken)
				server := myApp.Preferences().String(prefServer)
				if token == "" || server == "" {
					return fmt.Errorf("not logged in")
				}
				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return err
				}
				defer conn.Close()
				userClient := userpb.NewUserServiceClient(conn)
				_, err = userClient.ChangePassword(sessionCtx, &userpb.ChangePasswordRequest{OldPassword: oldPassword, NewPassword: newPassword})
				return err
			},
			func(bandID uint32) ([]ui.MemberInfo, error) {
				token := myApp.Preferences().String(prefToken)
				server := myApp.Preferences().String(prefServer)
				if token == "" || server == "" {
					return nil, fmt.Errorf("not logged in")
				}
				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return nil, err
				}
				defer conn.Close()
				bandClient := bandpb.NewBandServiceClient(conn)
				resp, err := bandClient.ListBandMembers(sessionCtx, &bandpb.ListBandMembersRequest{BandId: bandID})
				if err != nil {
					return nil, err
				}
				var members []ui.MemberInfo
				for _, m := range resp.GetMembers() {
					members = append(members, ui.MemberInfo{UserID: m.GetUserId(), Email: m.GetEmail(), Name: m.GetName(), Surname: m.GetSurname(), Role: m.GetRole()})
				}
				return members, nil
			},
			func(bandID uint32, userID uint32, email string) error {
				token := myApp.Preferences().String(prefToken)
				server := myApp.Preferences().String(prefServer)
				if token == "" || server == "" {
					return fmt.Errorf("not logged in")
				}
				conn, err := network.NewGRPCClient(server, token)
				if err != nil {
					return err
				}
				defer conn.Close()
				bandClient := bandpb.NewBandServiceClient(conn)
				_, err = bandClient.RemoveMember(sessionCtx, &bandpb.RemoveMemberRequest{BandId: bandID, UserId: userID, Email: email})
				return err
			},
		)
		mainWrapper.Objects = []fyne.CanvasObject{profileView}
		mainWrapper.Refresh()
	}

	appWithQuickSettings := ui.WrapWithQuickSettings(myWindow, myApp, mainWrapper, showLockScreen, onSwitchProfile, isCloudConnected)

	showDashboard()
	myWindow.SetContent(appWithQuickSettings)

	savedToken := myApp.Preferences().String(prefToken)
	savedServer := myApp.Preferences().String(prefServer)
	if savedToken != "" && savedServer != "" {
		startBackgroundSyncLoop(savedServer, savedToken)
	}
}
