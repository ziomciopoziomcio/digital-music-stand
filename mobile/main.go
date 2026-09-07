package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var AppVersion = "mobile-v0.2.0"

func main() {
	myApp := app.NewWithID("com.digitalmusicstand.pilot")
	myWindow := myApp.NewWindow("DMS Pilot")
	myWindow.Resize(fyne.NewSize(360, 800))

	var showLogin func()
	var showManualConnect func()
	var showPilot func(server, token, concertID, localIP, localPIN string)
	var autoDiscover func(server, token string)

	showLogin = func() {
		serverEntry := widget.NewEntry()
		serverEntry.SetText(myApp.Preferences().StringWithFallback("server", "localhost:50051"))
		emailEntry := widget.NewEntry()
		emailEntry.SetText(myApp.Preferences().String("email"))
		passEntry := widget.NewPasswordEntry()
		passEntry.SetPlaceHolder("Password")

		loginBtn := widget.NewButtonWithIcon("Login & Auto-Connect", theme.LoginIcon(), func() {
			server := serverEntry.Text
			myApp.Preferences().SetString("server", server)
			myApp.Preferences().SetString("email", emailEntry.Text)

			progress := dialog.NewCustomWithoutButtons("Authenticating...", container.NewPadded(widget.NewProgressBarInfinite()), myWindow)
			progress.Show()

			go func() {
				conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					progress.Hide()
					dialog.ShowError(err, myWindow)
					return
				}
				defer conn.Close()

				client := userpb.NewUserServiceClient(conn)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				resp, err := client.LoginUser(ctx, &userpb.LoginUserRequest{Email: emailEntry.Text, Password: passEntry.Text})
				progress.Hide()
				if err != nil {
					dialog.ShowInformation("Login Failed", "Invalid credentials or server unreachable.", myWindow)
					return
				}

				myApp.Preferences().SetString("token", resp.GetToken())
				autoDiscover(server, resp.GetToken())
			}()
		})
		loginBtn.Importance = widget.HighImportance

		offlineBtn := widget.NewButtonWithIcon("Offline / Manual Connect", theme.ComputerIcon(), func() {
			showManualConnect()
		})

		form := container.NewVScroll(container.NewVBox(
			widget.NewLabelWithStyle("DMS Pilot", fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Italic: true}),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Cloud Auto-Discovery:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			serverEntry, emailEntry, passEntry,
			widget.NewLabel(""),
			loginBtn,
			widget.NewSeparator(),
			widget.NewLabelWithStyle("No internet on stage?", fyne.TextAlignCenter, fyne.TextStyle{}),
			offlineBtn,
		))
		myWindow.SetContent(container.NewPadded(form))
	}

	showManualConnect = func() {
		ipEntry := widget.NewEntry()
		ipEntry.SetPlaceHolder("e.g. 192.168.1.50")
		ipEntry.SetText(myApp.Preferences().String("local_ip"))
		pinEntry := widget.NewEntry()
		pinEntry.SetPlaceHolder("4-digit PIN")
		pinEntry.SetText(myApp.Preferences().String("local_pin"))

		connectBtn := widget.NewButtonWithIcon("Connect via Wi-Fi", theme.MediaPlayIcon(), func() {
			myApp.Preferences().SetString("local_ip", ipEntry.Text)
			myApp.Preferences().SetString("local_pin", pinEntry.Text)
			showPilot("", "", "", ipEntry.Text, pinEntry.Text)
		})
		connectBtn.Importance = widget.HighImportance

		// POPRAWKA: Zmiana ikony na istniejącą w Fyne v2 (SearchIcon)
		qrBtn := widget.NewButtonWithIcon("Scan QR Code (Camera)", theme.SearchIcon(), func() {
			dialog.ShowInformation("QR Scanner", "Camera integration is coming soon. Use manual entry for now.", myWindow)
		})

		backBtn := widget.NewButtonWithIcon("Back to Login", theme.NavigateBackIcon(), showLogin)

		form := container.NewVBox(
			widget.NewLabelWithStyle("Manual Local Connection", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Check your tablet's 'Tools -> Mobile Pilot' for these details."),
			widget.NewSeparator(),
			widget.NewLabel("Tablet IP Address:"), ipEntry,
			widget.NewLabel("PIN Code:"), pinEntry,
			widget.NewLabel(""),
			connectBtn,
			qrBtn,
			layout.NewSpacer(),
			widget.NewSeparator(),
			backBtn,
		)
		myWindow.SetContent(container.NewPadded(form))
	}

	autoDiscover = func(server, token string) {
		progress := dialog.NewCustomWithoutButtons("Searching for active tablet...", container.NewPadded(widget.NewProgressBarInfinite()), myWindow)
		progress.Show()

		go func() {
			conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				progress.Hide()
				dialog.ShowError(err, myWindow)
				return
			}
			defer conn.Close()

			concertClient := concertpb.NewConcertServiceClient(conn)
			ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
			resp, err := concertClient.ListMyConcerts(ctx, &concertpb.ListMyConcertsRequest{})
			if err != nil || len(resp.GetConcerts()) == 0 {
				progress.Hide()
				dialog.ShowInformation("No Concerts", "You don't have any concerts to connect to.", myWindow)
				return
			}

			foundChan := make(chan []string, 1)
			var wg sync.WaitGroup

			for _, c := range resp.GetConcerts() {
				wg.Add(1)
				go func(cid string) {
					defer wg.Done()
					syncClient := syncpb.NewLiveSyncServiceClient(conn)
					streamCtx, cancel := context.WithCancel(ctx)
					defer cancel()

					stream, err := syncClient.SyncConcertStream(streamCtx)
					if err != nil {
						return
					}

					_ = stream.Send(&syncpb.SyncRequest{
						ConcertId: cid,
						Action:    syncpb.ActionType_UNKNOWN_ACTION,
						IsLeader:  false,
					})

					go func() {
						time.Sleep(2 * time.Second)
						cancel()
					}()

					for {
						msg, err := stream.Recv()
						if err != nil {
							return
						}
						if msg.GetAction() == syncpb.ActionType_BROADCAST_INFO {
							parts := strings.Split(msg.GetPayload(), "|")
							if len(parts) == 2 {
								select {
								case foundChan <- []string{cid, parts[0], parts[1]}:
								default:
								}
								return
							}
						}
					}
				}(c.Id)
			}

			go func() {
				wg.Wait()
				close(foundChan)
			}()

			select {
			case res := <-foundChan:
				progress.Hide()
				if res != nil {
					showPilot(server, token, res[0], res[1], res[2])
				} else {
					showManualConnect()
					dialog.ShowInformation("Not Found", "No active tablet found on the server. Connect manually.", myWindow)
				}
			}
		}()
	}

	showPilot = func(server, token, concertID, localIP, pin string) {
		infoLabel := widget.NewLabelWithStyle("Connecting...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		// POPRAWKA: Używamy canvas.Text do kolorowania zamiast widget.Label
		connLabel := canvas.NewText("Route: Unknown", theme.DisabledColor())
		connLabel.Alignment = fyne.TextAlignCenter
		connLabel.TextStyle = fyne.TextStyle{Italic: true}

		var syncConn *grpc.ClientConn
		var syncClient syncpb.LiveSyncServiceClient

		if server != "" && token != "" && concertID != "" {
			syncConn, _ = grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if syncConn != nil {
				syncClient = syncpb.NewLiveSyncServiceClient(syncConn)
				ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
				// Połączenie z chmurą
				syncClient.SyncConcertStream(ctx)
			}
		}

		httpClient := &http.Client{Timeout: 500 * time.Millisecond}

		sendCommand := func(actionStr string, grpcAction syncpb.ActionType) {
			if localIP != "" && pin != "" {
				url := fmt.Sprintf("http://%s:8089/api/command", localIP)
				payload, _ := json.Marshal(map[string]interface{}{"action": actionStr, "value": 0})
				req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
				req.Header.Set("X-Remote-PIN", pin)

				resp, err := httpClient.Do(req)
				if err == nil && resp.StatusCode == 200 {
					connLabel.Text = "Route: Local Wi-Fi (Fast)"
					connLabel.Color = theme.SuccessColor()
					connLabel.Refresh()
					return
				}
			}

			if syncClient != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

				stream, err := syncClient.SyncConcertStream(ctx)
				if err == nil {
					_ = stream.Send(&syncpb.SyncRequest{
						ConcertId: concertID,
						Action:    grpcAction,
						IsLeader:  true,
					})
					connLabel.Text = "Route: Cloud (Fallback)"
					connLabel.Color = theme.WarningColor()
					connLabel.Refresh()
					return
				}
			}

			connLabel.Text = "Route: Offline / Unreachable"
			connLabel.Color = theme.ErrorColor()
			connLabel.Refresh()
		}

		prevItemBtn := widget.NewButtonWithIcon("Prev Item", theme.MediaSkipPreviousIcon(), func() { sendCommand("PREV_ITEM", syncpb.ActionType_PREV_ITEM) })
		nextItemBtn := widget.NewButtonWithIcon("Next Item", theme.MediaSkipNextIcon(), func() { sendCommand("NEXT_ITEM", syncpb.ActionType_NEXT_ITEM) })
		prevPageBtn := widget.NewButtonWithIcon("Prev Page", theme.NavigateBackIcon(), func() { sendCommand("PREV_PAGE", syncpb.ActionType_PREV_PAGE) })
		nextPageBtn := widget.NewButtonWithIcon("Next Page", theme.NavigateNextIcon(), func() { sendCommand("NEXT_PAGE", syncpb.ActionType_NEXT_PAGE) })

		prevPageBtn.Importance = widget.HighImportance
		nextPageBtn.Importance = widget.HighImportance

		grid := container.NewGridWithColumns(2, prevItemBtn, nextItemBtn, prevPageBtn, nextPageBtn)

		toolsGrid := container.NewGridWithColumns(2,
			widget.NewButtonWithIcon("Timer", theme.HistoryIcon(), func() { sendCommand("TOGGLE_TIMER", syncpb.ActionType_TOGGLE_TIMER) }),
			widget.NewButtonWithIcon("Lock Tablet", theme.LogoutIcon(), func() { sendCommand("LOCK_SCREEN", syncpb.ActionType_UNKNOWN_ACTION) }),
		)

		go func() {
			for {
				time.Sleep(1 * time.Second)
				if localIP == "" {
					continue
				}

				req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s:8089/api/state", localIP), nil)
				req.Header.Set("X-Remote-PIN", pin)
				if resp, err := httpClient.Do(req); err == nil && resp.StatusCode == 200 {
					var state map[string]interface{}
					if json.NewDecoder(resp.Body).Decode(&state) == nil {
						name := state["concert_name"].(string)
						page := int(state["current_page"].(float64))
						total := int(state["total_pages"].(float64))
						infoLabel.SetText(fmt.Sprintf("%s\nPage %d / %d", name, page+1, total))
					}
				}
			}
		}()

		disconnectBtn := widget.NewButtonWithIcon("Disconnect", theme.CancelIcon(), func() {
			if syncConn != nil {
				syncConn.Close()
			}
			showLogin()
		})
		disconnectBtn.Importance = widget.DangerImportance

		layoutWrapper := container.NewVBox(
			infoLabel, connLabel,
			widget.NewSeparator(), widget.NewLabel(""),
			grid, widget.NewLabel(""), toolsGrid,
			layout.NewSpacer(), widget.NewSeparator(), disconnectBtn,
		)
		myWindow.SetContent(container.NewPadded(layoutWrapper))
	}

	showLogin()
	myWindow.ShowAndRun()
}
