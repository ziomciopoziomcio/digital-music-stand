package main

import (
	"bytes"
	"context"
	"crypto/tls"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/userpb"
)

type ScoreItem struct {
	ID      string
	Title   string
	OnLAN   bool
	OnCloud bool
}

type DiscoveredDevice struct {
	IP   string
	PIN  string
	Name string
}

func sanitizeAddress(addr string) string {
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	return strings.TrimRight(addr, "/")
}

func shouldUseTLS(serverAddr, mode string) bool {
	if mode == "ON" {
		return true
	}
	if mode == "OFF" {
		return false
	}
	return strings.HasSuffix(serverAddr, ":443") || strings.HasPrefix(serverAddr, "https://")
}

func getTransportCredentials(useTLS bool) credentials.TransportCredentials {
	if useTLS {
		return credentials.NewTLS(&tls.Config{})
	}
	return insecure.NewCredentials()
}

func main() {
	myApp := app.NewWithID("com.digitalmusicstand.mobile")
	myWindow := myApp.NewWindow("DMS Remote")
	myWindow.Resize(fyne.NewSize(360, 800))

	var showLogin func()
	var showMain func(server, token, localIP, localPIN string, tlsMode string)
	var viewScore func(ip, pin, server, token, scoreID, title string, startPage int, useLAN bool, tlsMode string)

	httpClient := &http.Client{Timeout: 3 * time.Second}

	showLogin = func() {
		serverEntry := widget.NewEntry()
		serverEntry.SetText(myApp.Preferences().StringWithFallback("server", "localhost:50051"))

		emailEntry := widget.NewEntry()
		emailEntry.SetText(myApp.Preferences().String("email"))

		passEntry := widget.NewPasswordEntry()

		tlsMode := myApp.Preferences().StringWithFallback("tls_mode", "AUTO")
		tlsSelect := widget.NewSelect([]string{"AUTO", "ON", "OFF"}, nil)
		tlsSelect.SetSelected(tlsMode)

		discoveredListContainer := container.NewVBox(widget.NewLabelWithStyle("Searching for local tablets...", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}))

		go func() {
			connUDP, err := net.ListenPacket("udp4", ":0")
			if err != nil {
				discoveredListContainer.Objects = []fyne.CanvasObject{widget.NewLabel("Network discovery error")}
				discoveredListContainer.Refresh()
				return
			}
			defer connUDP.Close()

			addr, _ := net.ResolveUDPAddr("udp4", "255.255.255.255:8090")
			connUDP.WriteTo([]byte("DMS_DISCOVER"), addr)
			connUDP.SetReadDeadline(time.Now().Add(2 * time.Second))

			devicesMap := make(map[string]DiscoveredDevice)

			for {
				buf := make([]byte, 1024)
				n, srvAddr, err := connUDP.ReadFrom(buf)
				if err != nil {
					break
				}
				ip := strings.Split(srvAddr.String(), ":")[0]
				payload := string(buf[:n])
				parts := strings.SplitN(payload, "|", 2)
				pin := parts[0]
				name := "Tablet (" + ip + ")"
				if len(parts) == 2 && parts[1] != "" {
					name = parts[1]
				}
				devicesMap[ip] = DiscoveredDevice{IP: ip, PIN: pin, Name: name}
			}

			if len(devicesMap) == 0 {
				discoveredListContainer.Objects = []fyne.CanvasObject{widget.NewLabelWithStyle("No local tablets found.", fyne.TextAlignCenter, fyne.TextStyle{})}
			} else {
				var items []fyne.CanvasObject
				for _, d := range devicesMap {
					dev := d
					btn := widget.NewButtonWithIcon(dev.Name, theme.ComputerIcon(), func() {
						pairProgress := dialog.NewCustomWithoutButtons("Waiting for tablet confirmation...", container.NewPadded(widget.NewProgressBarInfinite()), myWindow)
						pairProgress.Show()

						go func() {
							url := fmt.Sprintf("http://%s:8089/api/pair", dev.IP)
							req, _ := http.NewRequest("POST", url, nil)
							req.Header.Set("X-Remote-PIN", dev.PIN)
							pairClient := &http.Client{Timeout: 30 * time.Second}
							resp, err := pairClient.Do(req)

							pairProgress.Hide()

							if err == nil && resp.StatusCode == 200 {
								myApp.Preferences().SetString("local_ip", dev.IP)
								myApp.Preferences().SetString("local_pin", dev.PIN)
								showMain("", "", dev.IP, dev.PIN, tlsSelect.Selected)
							} else {
								dialog.ShowInformation("Pairing Rejected", "The tablet rejected the connection or timed out.", myWindow)
							}
						}()
					})
					items = append(items, btn)
				}
				discoveredListContainer.Objects = items
			}
			discoveredListContainer.Refresh()
		}()

		loginBtn := widget.NewButtonWithIcon("Login to Cloud", theme.LoginIcon(), func() {
			server := serverEntry.Text
			selectedTlsMode := tlsSelect.Selected

			myApp.Preferences().SetString("server", server)
			myApp.Preferences().SetString("email", emailEntry.Text)
			myApp.Preferences().SetString("tls_mode", selectedTlsMode)

			progress := dialog.NewCustomWithoutButtons("Authenticating...", container.NewPadded(widget.NewProgressBarInfinite()), myWindow)
			progress.Show()

			go func() {
				cleanServer := sanitizeAddress(server)
				useTLS := shouldUseTLS(server, selectedTlsMode)
				creds := getTransportCredentials(useTLS)

				conn, err := grpc.NewClient(cleanServer, grpc.WithTransportCredentials(creds))
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
				showMain(server, resp.GetToken(), "", "", selectedTlsMode)
			}()
		})
		loginBtn.Importance = widget.HighImportance

		skipBtn := widget.NewButton("Skip & Work Offline", func() {
			showMain("", "", "", "", tlsSelect.Selected)
		})

		form := container.NewVScroll(container.NewVBox(
			widget.NewLabelWithStyle("Digital Music Stand", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Local Network Discovery", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			discoveredListContainer,
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Cloud Connection", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Server:"),
			serverEntry,
			widget.NewLabel("TLS Mode:"),
			tlsSelect,
			widget.NewLabel("Email:"),
			emailEntry,
			widget.NewLabel("Password:"),
			passEntry,
			widget.NewLabel(""),
			loginBtn,
			widget.NewLabel(""),
			skipBtn,
		))

		myWindow.SetContent(container.NewPadded(form))
	}

	viewScore = func(ip, pin, server, token, scoreID, title string, startPage int, useLAN bool, tlsMode string) {
		page := startPage
		imgCanvas := canvas.NewImageFromResource(nil)
		imgCanvas.FillMode = canvas.ImageFillContain

		pageLabel := widget.NewLabelWithStyle("Loading...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		var pdfMgr *pdf.Manager
		var totalPages int

		loadPage := func() {
			if useLAN {
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
				cleanServer := sanitizeAddress(server)
				useTLS := shouldUseTLS(server, tlsMode)
				creds := getTransportCredentials(useTLS)

				conn, _ := grpc.NewClient(cleanServer, grpc.WithTransportCredentials(creds))
				defer conn.Close()
				client := scorepb.NewScoreServiceClient(conn)
				ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)
				stream, err := client.DownloadScore(ctx, &scorepb.DownloadScoreRequest{ScoreId: scoreID})
				if err != nil {
					progress.Hide()
					showMain(server, token, ip, pin, tlsMode)
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
					showMain(server, token, ip, pin, tlsMode)
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

		closeBtn := widget.NewButtonWithIcon("Back", theme.CancelIcon(), func() {
			if pdfMgr != nil {
				pdfMgr.Close()
				os.Remove(filepath.Join(os.TempDir(), scoreID+".pdf"))
			}
			showMain(server, token, ip, pin, tlsMode)
		})

		topBar := container.NewBorder(nil, nil, closeBtn, nil, widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		bottomBar := container.NewBorder(nil, nil, prevBtn, nextBtn, pageLabel)

		myWindow.SetContent(container.NewBorder(topBar, bottomBar, nil, nil, imgCanvas))
	}

	showMain = func(server, token, localIP, localPIN string, tlsMode string) {
		var syncConn *grpc.ClientConn
		var syncClient syncpb.LiveSyncServiceClient
		var activeConcertID string

		fastClient := &http.Client{Timeout: 500 * time.Millisecond}
		stopPolling := make(chan struct{})

		infoLabel := widget.NewLabelWithStyle("Fetching state...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		pageInfoLabel := widget.NewLabelWithStyle("-", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

		connStatusLabel := canvas.NewText("Connecting...", theme.DisabledColor())
		connStatusLabel.Alignment = fyne.TextAlignCenter

		if server != "" && token != "" {
			cleanServer := sanitizeAddress(server)
			useTLS := shouldUseTLS(server, tlsMode)
			creds := getTransportCredentials(useTLS)

			syncConn, _ = grpc.NewClient(cleanServer, grpc.WithTransportCredentials(creds))
			if syncConn != nil {
				syncClient = syncpb.NewLiveSyncServiceClient(syncConn)
			}
		}

		sendCommand := func(actionStr string, grpcAction syncpb.ActionType, payloadStr string, val int) {
			if localIP != "" && localPIN != "" {
				url := fmt.Sprintf("http://%s:8089/api/command", localIP)
				payload, _ := json.Marshal(map[string]interface{}{"action": actionStr, "value": val, "payload": payloadStr})
				req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
				req.Header.Set("X-Remote-PIN", localPIN)
				resp, err := fastClient.Do(req)
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
					targetPayload := localPIN
					if payloadStr != "" {
						targetPayload = localPIN + "|" + payloadStr
					}

					_ = stream.Send(&syncpb.SyncRequest{
						ConcertId: activeConcertID,
						Action:    grpcAction,
						IsLeader:  false,
						Payload:   targetPayload,
						ItemIndex: uint32(val),
					})
				}
			}
		}

		prevPageBtn := widget.NewButtonWithIcon("Previous Page", theme.NavigateBackIcon(), func() {
			sendCommand("PREV_PAGE", syncpb.ActionType_PREV_PAGE, "", 0)
		})
		nextPageBtn := widget.NewButtonWithIcon("Next Page", theme.NavigateNextIcon(), func() {
			sendCommand("NEXT_PAGE", syncpb.ActionType_NEXT_PAGE, "", 0)
		})

		timerBtn := widget.NewButtonWithIcon("Play/Pause Timer", theme.HistoryIcon(), func() {
			sendCommand("TOGGLE_TIMER", syncpb.ActionType_TOGGLE_TIMER, "", 0)
		})

		lockBtn := widget.NewButtonWithIcon("Lock Tablet", theme.LogoutIcon(), nil)

		var setlistTitles []string
		var currentItemIdx int = -1
		var currentScoreID string
		var currentPage int

		setlistList := widget.NewList(
			func() int { return len(setlistTitles) },
			func() fyne.CanvasObject {
				return container.NewPadded(container.NewHBox(
					widget.NewIcon(theme.DocumentIcon()),
					widget.NewLabel("Template"),
				))
			},
			func(i widget.ListItemID, o fyne.CanvasObject) {
				box := o.(*fyne.Container).Objects[0].(*fyne.Container)
				icon := box.Objects[0].(*widget.Icon)
				label := box.Objects[1].(*widget.Label)
				label.SetText(fmt.Sprintf("%d. %s", i+1, setlistTitles[i]))
				if int(i) < currentItemIdx {
					icon.SetResource(theme.ConfirmIcon())
					label.TextStyle = fyne.TextStyle{Italic: true}
				} else if int(i) == currentItemIdx {
					icon.SetResource(theme.MediaPlayIcon())
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else {
					icon.SetResource(theme.DocumentIcon())
					label.TextStyle = fyne.TextStyle{}
				}
				label.Refresh()
			},
		)

		setlistList.OnSelected = func(id widget.ListItemID) {
			setlistList.Unselect(id)
			sendCommand("JUMP_TO_ITEM", syncpb.ActionType_JUMP_TO_ITEM, "", int(id))
		}

		previewBtn := widget.NewButtonWithIcon("Preview Current Page", theme.VisibilityIcon(), func() {
			if currentScoreID != "" {
				close(stopPolling)
				viewScore(localIP, localPIN, server, token, currentScoreID, "Preview", currentPage, localIP != "", tlsMode)
			}
		})

		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopPolling:
					return
				case <-ticker.C:
					connected := false
					if localIP != "" && localPIN != "" {
						url := fmt.Sprintf("http://%s:8089/api/state", localIP)
						req, _ := http.NewRequest("GET", url, nil)
						req.Header.Set("X-Remote-PIN", localPIN)
						if resp, err := httpClient.Do(req); err == nil && resp.StatusCode == 200 {
							var state map[string]interface{}
							if json.NewDecoder(resp.Body).Decode(&state) == nil {
								if cid, ok := state["concert_id"].(string); ok {
									activeConcertID = cid
								}
								name := state["concert_name"].(string)
								page := int(state["current_page"].(float64))
								total := int(state["total_pages"].(float64))
								isLocked, _ := state["is_locked"].(bool)
								timerSec, _ := state["timer_seconds"].(float64)

								currentScoreID, _ = state["current_score_id"].(string)
								currentPage = page

								var newList []string
								if rawList, ok := state["setlist"].([]interface{}); ok {
									for _, v := range rawList {
										newList = append(newList, v.(string))
									}
								}

								newIdx := int(state["current_item_idx"].(float64))

								setlistChanged := false
								if len(newList) != len(setlistTitles) {
									setlistChanged = true
								} else {
									for i := range newList {
										if newList[i] != setlistTitles[i] {
											setlistChanged = true
											break
										}
									}
								}

								if setlistChanged || newIdx != currentItemIdx {
									setlistTitles = newList
									currentItemIdx = newIdx
									setlistList.Refresh()
								}

								if name == "" {
									infoLabel.SetText("Tablet is idle")
									pageInfoLabel.SetText("")
								} else {
									infoLabel.SetText(name)
									if total > 0 {
										pageInfoLabel.SetText(fmt.Sprintf("Page %d of %d", page+1, total))
									} else {
										pageInfoLabel.SetText(fmt.Sprintf("Timer: %02d:%02d", int(timerSec)/60, int(timerSec)%60))
									}

									if isLocked {
										lockBtn.SetText("Unlock Tablet")
										lockBtn.SetIcon(theme.LoginIcon())
										lockBtn.OnTapped = func() {
											pinEntry := widget.NewPasswordEntry()
											dialog.ShowCustomConfirm("Unlock Tablet", "Unlock", "Cancel", pinEntry, func(ok bool) {
												if ok {
													sendCommand("UNLOCK_SCREEN", syncpb.ActionType_UNLOCK_SCREEN, pinEntry.Text, 0)
												}
											}, myWindow)
										}
										prevPageBtn.Disable()
										nextPageBtn.Disable()
									} else {
										lockBtn.SetText("Lock Tablet")
										lockBtn.SetIcon(theme.LogoutIcon())
										lockBtn.OnTapped = func() {
											sendCommand("LOCK_SCREEN", syncpb.ActionType_LOCK_SCREEN, "", 0)
										}
										prevPageBtn.Enable()
										nextPageBtn.Enable()
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
			infoLabel,
			pageInfoLabel,
			widget.NewSeparator(),
			widget.NewLabel(""),
			container.NewGridWithColumns(2, prevPageBtn, nextPageBtn),
			widget.NewLabel(""),
			container.NewGridWithColumns(2, timerBtn, lockBtn),
			previewBtn,
			layout.NewSpacer(),
			container.NewCenter(connStatusLabel),
		)

		libraryList := widget.NewList(
			func() int { return 0 },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {},
		)

		libraryContent := container.NewBorder(
			widget.NewButtonWithIcon("Refresh Library", theme.ViewRefreshIcon(), func() {
				go func() {
					scoresMap := make(map[string]ScoreItem)
					if server != "" && token != "" {
						cleanServer := sanitizeAddress(server)
						useTLS := shouldUseTLS(server, tlsMode)
						creds := getTransportCredentials(useTLS)

						conn, err := grpc.NewClient(cleanServer, grpc.WithTransportCredentials(creds))
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
					libraryList.CreateItem = func() fyne.CanvasObject {
						return widget.NewButton("", nil)
					}
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
							close(stopPolling)
							useLAN := s.OnLAN && localIP != ""
							viewScore(localIP, localPIN, server, token, s.ID, s.Title, 0, useLAN, tlsMode)
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
			container.NewTabItemWithIcon("Setlist", theme.ListIcon(), container.NewPadded(setlistList)),
			container.NewTabItemWithIcon("Library", theme.DocumentIcon(), container.NewPadded(libraryContent)),
		)

		disconnectBtn := widget.NewButtonWithIcon("Disconnect & Forget", theme.LogoutIcon(), func() {
			close(stopPolling)
			if syncConn != nil {
				syncConn.Close()
			}
			myApp.Preferences().RemoveValue("token")
			myApp.Preferences().RemoveValue("local_ip")
			myApp.Preferences().RemoveValue("local_pin")
			showLogin()
		})
		disconnectBtn.Importance = widget.DangerImportance

		myWindow.SetContent(container.NewBorder(nil, disconnectBtn, nil, nil, tabs))
	}

	localIP := myApp.Preferences().String("local_ip")
	localPIN := myApp.Preferences().String("local_pin")
	token := myApp.Preferences().String("token")
	server := myApp.Preferences().String("server")
	tlsMode := myApp.Preferences().StringWithFallback("tls_mode", "AUTO")

	if (localIP != "" && localPIN != "") || (token != "" && server != "") {
		showMain(server, token, localIP, localPIN, tlsMode)
	} else {
		showLogin()
	}

	myWindow.ShowAndRun()
}
