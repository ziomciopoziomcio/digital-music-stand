package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type ScoreItem struct {
	ID      string
	Title   string
	OnLAN   bool
	OnCloud bool
}

func main() {
	myApp := app.NewWithID("com.digitalmusicstand.mobile")
	myWindow := myApp.NewWindow("DMS Remote")
	myWindow.Resize(fyne.NewSize(360, 800))

	var showLogin func()
	var showMain func(server, token string)
	var viewScore func(ip, pin, server, token, scoreID, title string, useLAN bool)

	showLogin = func() {
		serverEntry := widget.NewEntry()
		serverEntry.SetText(myApp.Preferences().StringWithFallback("server", "localhost:50051"))
		emailEntry := widget.NewEntry()
		emailEntry.SetText(myApp.Preferences().String("email"))
		passEntry := widget.NewPasswordEntry()

		loginBtn := widget.NewButtonWithIcon("Login", theme.LoginIcon(), func() {
			server := serverEntry.Text
			myApp.Preferences().SetString("server", server)
			myApp.Preferences().SetString("email", emailEntry.Text)

			progress := dialog.NewCustomWithoutButtons("", container.NewPadded(widget.NewProgressBarInfinite()), myWindow)
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

				resp, err := client.LoginUser(ctx, &userpb.LoginUserRequest{
					Email:    emailEntry.Text,
					Password: passEntry.Text,
				})
				progress.Hide()

				if err != nil {
					dialog.ShowError(err, myWindow)
					return
				}

				myApp.Preferences().SetString("token", resp.GetToken())
				showMain(server, resp.GetToken())
			}()
		})
		loginBtn.Importance = widget.HighImportance

		skipBtn := widget.NewButton("Skip Login (LAN Only)", func() {
			showMain("", "")
		})

		form := container.NewVScroll(container.NewVBox(
			widget.NewLabelWithStyle("Digital Music Stand", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabel("Server:"), serverEntry,
			widget.NewLabel("Email:"), emailEntry,
			widget.NewLabel("Password:"), passEntry,
			widget.NewLabel(""),
			loginBtn,
			widget.NewLabel(""),
			skipBtn,
		))
		myWindow.SetContent(container.NewPadded(form))
	}

	viewScore = func(ip, pin, server, token, scoreID, title string, useLAN bool) {
		page := 0
		imgCanvas := canvas.NewImageFromResource(nil)
		imgCanvas.FillMode = canvas.ImageFillContain
		pageLabel := widget.NewLabelWithStyle("Loading...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		var pdfMgr *pdf.Manager
		var totalPages int

		loadPage := func() {
			if useLAN {
				httpClient := &http.Client{Timeout: 3 * time.Second}
				url := fmt.Sprintf("http://%s:8089/api/render?id=%s&page=%d", ip, scoreID, page)
				req, _ := http.NewRequest("GET", url, nil)
				req.Header.Set("X-Remote-PIN", pin)

				resp, err := httpClient.Do(req)
				if err != nil || resp.StatusCode != 200 {
					if page > 0 {
						page--
					}
					return
				}
				defer resp.Body.Close()
				data, _ := io.ReadAll(resp.Body)
				imgCanvas.Resource = fyne.NewStaticResource("page.png", data)
				imgCanvas.Refresh()
				pageLabel.SetText(fmt.Sprintf("Page %d", page+1))
			} else {
				if pdfMgr != nil && totalPages > 0 {
					img, err := pdfMgr.GetPageImage(page)
					if err == nil {
						imgCanvas.Image = img
						imgCanvas.Refresh()
						pageLabel.SetText(fmt.Sprintf("Page %d of %d", page+1, totalPages))
					}
				}
			}
		}

		if !useLAN && server != "" && token != "" {
			progress := dialog.NewCustomWithoutButtons("Downloading...", container.NewPadded(widget.NewProgressBarInfinite()), myWindow)
			progress.Show()

			go func() {
				conn, _ := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
				defer conn.Close()

				client := scorepb.NewScoreServiceClient(conn)
				ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
				stream, err := client.DownloadScore(ctx, &scorepb.DownloadScoreRequest{ScoreId: scoreID})
				if err != nil {
					progress.Hide()
					showMain(server, token)
					return
				}

				tmpFile := filepath.Join(os.TempDir(), scoreID+".pdf")
				f, _ := os.Create(tmpFile)
				for {
					chunk, err := stream.Recv()
					if err == io.EOF {
						break
					}
					f.Write(chunk.GetChunkData())
				}
				f.Close()

				mgr, err := pdf.NewManager(tmpFile)
				progress.Hide()

				if err == nil {
					pdfMgr = mgr
					totalPages = mgr.GetPageCount()
					loadPage()
				} else {
					showMain(server, token)
				}
			}()
		} else {
			loadPage()
		}

		prevBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
			if page > 0 {
				page--
				loadPage()
			}
		})
		nextBtn := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
			if !useLAN {
				if page < totalPages-1 {
					page++
					loadPage()
				}
			} else {
				page++
				loadPage()
			}
		})
		closeBtn := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
			if pdfMgr != nil {
				pdfMgr.Close()
				os.Remove(filepath.Join(os.TempDir(), scoreID+".pdf"))
			}
			showMain(server, token)
		})

		topBar := container.NewBorder(nil, nil, closeBtn, nil, widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		bottomBar := container.NewBorder(nil, nil, prevBtn, nextBtn, pageLabel)

		myWindow.SetContent(container.NewBorder(topBar, bottomBar, nil, nil, imgCanvas))
	}

	showMain = func(server, token string) {
		var localIP, localPIN string
		var syncConn *grpc.ClientConn
		var syncClient syncpb.LiveSyncServiceClient
		var activeConcertID string

		httpClient := &http.Client{Timeout: 500 * time.Millisecond}
		stopBackground := make(chan struct{})

		infoLabel := widget.NewLabelWithStyle("Searching for tablet...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		pageInfoLabel := widget.NewLabelWithStyle("-", fyne.TextAlignCenter, fyne.TextStyle{})
		connStatusLabel := canvas.NewText("Link: Offline", theme.ErrorColor())
		connStatusLabel.Alignment = fyne.TextAlignCenter

		if server != "" && token != "" {
			syncConn, _ = grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if syncConn != nil {
				syncClient = syncpb.NewLiveSyncServiceClient(syncConn)
			}
		}

		sendCommand := func(actionStr string, grpcAction syncpb.ActionType) {
			if localIP != "" && localPIN != "" {
				url := fmt.Sprintf("http://%s:8089/api/command", localIP)
				payload, _ := json.Marshal(map[string]interface{}{"action": actionStr, "value": 0})
				req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
				req.Header.Set("X-Remote-PIN", localPIN)

				resp, err := httpClient.Do(req)
				if err == nil && resp.StatusCode == 200 {
					return
				}
			}

			if syncClient != nil && activeConcertID != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

				stream, err := syncClient.SyncConcertStream(ctx)
				if err == nil {
					_ = stream.Send(&syncpb.SyncRequest{
						ConcertId: activeConcertID,
						Action:    grpcAction,
						IsLeader:  true,
					})
				}
			}
		}

		go func() {
			udpTicker := time.NewTicker(3 * time.Second)
			pollTicker := time.NewTicker(1 * time.Second)
			defer udpTicker.Stop()
			defer pollTicker.Stop()

			for {
				select {
				case <-stopBackground:
					return
				case <-udpTicker.C:
					connUDP, err := net.ListenPacket("udp4", ":0")
					if err == nil {
						addr, _ := net.ResolveUDPAddr("udp4", "255.255.255.255:8090")
						connUDP.WriteTo([]byte("DMS_DISCOVER"), addr)
						connUDP.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
						buf := make([]byte, 1024)
						n, srvAddr, err := connUDP.ReadFrom(buf)
						if err == nil {
							localIP = strings.Split(srvAddr.String(), ":")[0]
							localPIN = string(buf[:n])
						}
						connUDP.Close()
					}
				case <-pollTicker.C:
					connected := false
					if localIP != "" && localPIN != "" {
						url := fmt.Sprintf("http://%s:8089/api/state", localIP)
						req, _ := http.NewRequest("GET", url, nil)
						req.Header.Set("X-Remote-PIN", localPIN)

						if resp, err := httpClient.Do(req); err == nil && resp.StatusCode == 200 {
							var state map[string]interface{}
							if json.NewDecoder(resp.Body).Decode(&state) == nil {
								name := state["concert_name"].(string)
								page := int(state["current_page"].(float64))
								total := int(state["total_pages"].(float64))

								if name == "" {
									infoLabel.SetText("Tablet is idle")
									pageInfoLabel.SetText("")
								} else {
									infoLabel.SetText(name)
									if total > 0 {
										pageInfoLabel.SetText(fmt.Sprintf("Page %d of %d", page+1, total))
									} else {
										pageInfoLabel.SetText("Break / No Score")
									}
								}
								connStatusLabel.Text = "Link: Wi-Fi"
								connStatusLabel.Color = theme.SuccessColor()
								connStatusLabel.Refresh()
								connected = true
							}
						}
					}

					if !connected {
						if server != "" && token != "" {
							connStatusLabel.Text = "Link: Cloud Ready"
							connStatusLabel.Color = theme.WarningColor()
							connStatusLabel.Refresh()
							infoLabel.SetText("Waiting for stage data...")
							pageInfoLabel.SetText("")
						} else {
							connStatusLabel.Text = "Link: Offline"
							connStatusLabel.Color = theme.ErrorColor()
							connStatusLabel.Refresh()
							infoLabel.SetText("No connection")
							pageInfoLabel.SetText("")
						}
					}
				}
			}
		}()

		remoteContent := container.NewVBox(
			infoLabel, pageInfoLabel,
			widget.NewSeparator(),
			widget.NewLabel(""),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Previous Item", theme.MediaSkipPreviousIcon(), func() { sendCommand("PREV_ITEM", syncpb.ActionType_PREV_ITEM) }),
				widget.NewButtonWithIcon("Next Item", theme.MediaSkipNextIcon(), func() { sendCommand("NEXT_ITEM", syncpb.ActionType_NEXT_ITEM) }),
				widget.NewButtonWithIcon("Previous Page", theme.NavigateBackIcon(), func() { sendCommand("PREV_PAGE", syncpb.ActionType_PREV_PAGE) }),
				widget.NewButtonWithIcon("Next Page", theme.NavigateNextIcon(), func() { sendCommand("NEXT_PAGE", syncpb.ActionType_NEXT_PAGE) }),
			),
			widget.NewLabel(""),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Play/Pause Timer", theme.HistoryIcon(), func() { sendCommand("TOGGLE_TIMER", syncpb.ActionType_TOGGLE_TIMER) }),
				widget.NewButtonWithIcon("Lock Screen", theme.LogoutIcon(), func() { sendCommand("LOCK_SCREEN", syncpb.ActionType_UNKNOWN_ACTION) }),
			),
			layout.NewSpacer(),
			container.NewCenter(connStatusLabel),
		)

		libraryList := widget.NewList(
			func() int { return 0 },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {},
		)

		libraryContent := container.NewBorder(
			widget.NewButtonWithIcon("Refresh Scores", theme.ViewRefreshIcon(), func() {
				go func() {
					scoresMap := make(map[string]ScoreItem)

					if server != "" && token != "" {
						conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
						if err == nil {
							client := scorepb.NewScoreServiceClient(conn)
							ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
							if resp, err := client.ListMyScores(ctx, &scorepb.ListMyScoresRequest{}); err == nil {
								for _, s := range resp.GetScores() {
									scoresMap[s.Id] = ScoreItem{ID: s.Id, Title: s.Name, OnCloud: true}
								}
							}
							conn.Close()
						}
					}

					if localIP != "" && localPIN != "" {
						url := fmt.Sprintf("http://%s:8089/api/scores", localIP)
						req, _ := http.NewRequest("GET", url, nil)
						req.Header.Set("X-Remote-PIN", localPIN)
						if resp, err := httpClient.Do(req); err == nil && resp.StatusCode == 200 {
							var localScores []map[string]interface{}
							if json.NewDecoder(resp.Body).Decode(&localScores) == nil {
								for _, ls := range localScores {
									id := ls["ID"].(string)
									title := ls["Title"].(string)
									if alias, ok := ls["LocalAlias"].(string); ok && alias != "" {
										title = alias
									}
									if existing, exists := scoresMap[id]; exists {
										existing.OnLAN = true
										existing.Title = title
										scoresMap[id] = existing
									} else {
										scoresMap[id] = ScoreItem{ID: id, Title: title, OnLAN: true}
									}
								}
							}
						}
					}

					var fetched []ScoreItem
					for _, v := range scoresMap {
						fetched = append(fetched, v)
					}

					libraryList.Length = func() int { return len(fetched) }
					libraryList.CreateItem = func() fyne.CanvasObject { return widget.NewButton("", nil) }
					libraryList.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
						btn := o.(*widget.Button)
						s := fetched[i]

						icon := theme.DocumentIcon()
						if s.OnLAN {
							icon = theme.ComputerIcon()
						} else if s.OnCloud {
							icon = theme.StorageIcon()
						}

						btn.SetText(s.Title)
						btn.SetIcon(icon)
						btn.OnTapped = func() {
							close(stopBackground)
							useLAN := s.OnLAN && localIP != ""
							viewScore(localIP, localPIN, server, token, s.ID, s.Title, useLAN)
						}
					}
					libraryList.Refresh()
				}()
			}),
			nil, nil, nil,
			libraryList,
		)

		tabs := container.NewAppTabs(
			container.NewTabItemWithIcon("Control", theme.SettingsIcon(), container.NewPadded(remoteContent)),
			container.NewTabItemWithIcon("Library", theme.DocumentIcon(), container.NewPadded(libraryContent)),
		)

		logoutBtn := widget.NewButtonWithIcon("Logout", theme.LogoutIcon(), func() {
			close(stopBackground)
			if syncConn != nil {
				syncConn.Close()
			}
			myApp.Preferences().SetString("token", "")
			showLogin()
		})
		logoutBtn.Importance = widget.DangerImportance

		myWindow.SetContent(container.NewBorder(nil, logoutBtn, nil, nil, tabs))
	}

	showLogin()
	myWindow.ShowAndRun()
}
