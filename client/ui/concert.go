package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"google.golang.org/grpc"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
)

func BuildConcertMode(w fyne.Window, app fyne.App, db *localdb.DBManager, goBack func(), openSetup func(editingConcert *localdb.Concert), onDeleteConcert func()) *fyne.Container {
	contentWrapper := container.NewMax()

	var showConcertList func()
	var updateGrid func()
	var playConcert func(concert localdb.Concert)

	gridWrapper := container.NewMax()

	searchEntry := NewAutoKeyboardEntry()
	searchEntry.SetPlaceHolder("Search concerts (min. 3 chars)...")

	updateGrid = func() {
		isLoggedIn := app.Preferences().String("jwt_token") != ""
		concerts, err := db.GetConcerts()
		if err != nil {
			dialog.ShowError(err, w)
			concerts = []localdb.Concert{}
		}

		grid := container.NewGridWrap(fyne.NewSize(280, 220))
		query := strings.ToLower(searchEntry.Text)

		for _, c := range concerts {
			concert := c

			if len(query) >= 3 && !strings.Contains(strings.ToLower(concert.DisplayName()), query) {
				continue
			}

			nameLabel := widget.NewLabelWithStyle(concert.DisplayName(), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			detailsLabel := widget.NewLabelWithStyle(fmt.Sprintf("%s\n%s\nItems: %d", concert.Location, concert.StartTime, len(concert.Items)), fyne.TextAlignCenter, fyne.TextStyle{})

			openBtn := widget.NewButtonWithIcon("ENTER", theme.MediaPlayIcon(), func() {
				playConcert(concert)
			})
			openBtn.Importance = widget.HighImportance

			var actionButtons *fyne.Container

			if concert.IsOwner {
				shareBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
					ShowAccessDialog(w, app, "Share Concert", concert.Name, "Share", true, func(email *string, bandID *uint32, canEdit bool) error {
						token := app.Preferences().String("jwt_token")
						server := app.Preferences().String("server_addr")
						conn, err := network.NewGRPCClient(server, token)
						if err != nil {
							return err
						}
						defer conn.Close()

						client := concertpb.NewConcertServiceClient(conn)
						_, err = client.ShareConcert(context.Background(), &concertpb.ShareConcertRequest{
							ConcertId:    concert.ID,
							TargetEmail:  email,
							TargetBandId: bandID,
							CanEdit:      canEdit,
						})
						return err
					})
				})

				revokeBtn := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() {
					ShowAccessDialog(w, app, "Revoke Concert Access", concert.Name, "Revoke", false, func(email *string, bandID *uint32, canEdit bool) error {
						token := app.Preferences().String("jwt_token")
						server := app.Preferences().String("server_addr")
						conn, err := network.NewGRPCClient(server, token)
						if err != nil {
							return err
						}
						defer conn.Close()

						client := concertpb.NewConcertServiceClient(conn)
						_, err = client.RevokeConcertAccess(context.Background(), &concertpb.RevokeConcertAccessRequest{
							ConcertId:    concert.ID,
							TargetEmail:  email,
							TargetBandId: bandID,
						})
						return err
					})
				})
				revokeBtn.Importance = widget.DangerImportance

				if !isLoggedIn {
					shareBtn.Disable()
					revokeBtn.Disable()
				}

				editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
					openSetup(&concert)
				})

				deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
					dialog.ShowConfirm("Delete Concert", fmt.Sprintf("Are you sure you want to delete '%s'?", concert.Name), func(confirmed bool) {
						if confirmed {
							_ = db.MarkConcertDeleted(concert.ID)
							onDeleteConcert()
							updateGrid()
						}
					}, w)
				})
				deleteBtn.Importance = widget.DangerImportance

				actionButtons = container.NewHBox(shareBtn, revokeBtn, editBtn, deleteBtn, openBtn)
			} else if concert.CanEdit {
				sharedBadge := widget.NewLabelWithStyle("Shared", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

				setAliasBtn := widget.NewButtonWithIcon("Alias", theme.SettingsIcon(), func() {
					entry := NewAutoKeyboardEntry()
					entry.SetText(concert.DisplayName())
					var d dialog.Dialog
					formContent := container.NewVBox(
						widget.NewLabel("Set local alias (only visible to you):"),
						entry,
						container.NewHBox(
							layout.NewSpacer(),
							widget.NewButton("Save", func() {
								_ = db.SetConcertAlias(concert.ID, entry.Text)
								updateGrid()
								d.Hide()
							}),
							widget.NewButton("Clear", func() {
								_ = db.SetConcertAlias(concert.ID, "")
								updateGrid()
								d.Hide()
							}),
							widget.NewButton("Cancel", func() { d.Hide() }),
						),
					)
					d = dialog.NewCustomWithoutButtons("Set Alias", formContent, w)
					d.Show()
					w.Canvas().Focus(entry)
				})

				editBtn := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
					openSetup(&concert)
				})

				actionButtons = container.NewHBox(sharedBadge, setAliasBtn, editBtn, openBtn)
			} else {
				sharedBadge := widget.NewLabelWithStyle("Shared", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
				setAliasBtn := widget.NewButtonWithIcon("Alias", theme.SettingsIcon(), func() {
					entry := NewAutoKeyboardEntry()
					entry.SetText(concert.DisplayName())
					var d dialog.Dialog
					formContent := container.NewVBox(
						widget.NewLabel("Set local alias (only visible to you):"),
						entry,
						container.NewHBox(
							layout.NewSpacer(),
							widget.NewButton("Save", func() {
								_ = db.SetConcertAlias(concert.ID, entry.Text)
								updateGrid()
								d.Hide()
							}),
							widget.NewButton("Clear", func() {
								_ = db.SetConcertAlias(concert.ID, "")
								updateGrid()
								d.Hide()
							}),
							widget.NewButton("Cancel", func() { d.Hide() }),
						),
					)
					d = dialog.NewCustomWithoutButtons("Set Alias", formContent, w)
					d.Show()
					w.Canvas().Focus(entry)
				})
				actionButtons = container.NewHBox(sharedBadge, setAliasBtn, openBtn)
			}

			cardContent := container.NewVBox(nameLabel, detailsLabel, layout.NewSpacer())
			card := widget.NewCard("", "", cardContent)

			item := container.NewBorder(nil, actionButtons, nil, nil, card)
			grid.Add(item)
		}

		gridWrapper.Objects = []fyne.CanvasObject{container.NewPadded(container.NewVScroll(grid))}
		gridWrapper.Refresh()
	}

	searchEntry.OnChanged = func(s string) {
		if len(s) >= 3 || len(s) == 0 {
			updateGrid()
		}
	}

	showConcertList = func() {
		updateGrid()
		backBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backBtn.Importance = widget.WarningImportance

		newConcertBtn := widget.NewButtonWithIcon("New Concert", theme.ContentAddIcon(), func() {
			openSetup(nil)
		})
		newConcertBtn.Importance = widget.HighImportance
		topControls := container.NewHBox(newConcertBtn)

		searchContainer := container.NewPadded(searchEntry)
		header := container.NewBorder(nil, nil, backBtn, topControls, searchContainer)
		view := container.NewBorder(header, nil, nil, nil, gridWrapper)

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	playConcert = func(concert localdb.Concert) {
		if len(concert.Items) == 0 {
			dialog.ShowInformation("Empty Setlist", "This concert has no items assigned.", w)
			return
		}

		var syncConn *grpc.ClientConn
		var syncStream syncpb.LiveSyncService_SyncConcertStreamClient
		var syncCtx context.Context
		var syncCancel context.CancelFunc

		var isConnected bool
		var syncMode int
		var leaderAvailable bool
		var previewMode bool
		var autoFollow bool

		var leaderItemIdx int
		var leaderPage int
		var leaderTimer int

		var savedItemIdx int
		var savedPage int
		var savedTimer int

		var timerClockLabel *canvas.Text
		var timerStatusLabel *widget.Label
		var startPauseBtn *widget.Button
		var isTimerRunning bool
		var remainingSec int

		var startSyncBackground func()
		var sendStateUpdate func()
		var updateSyncUI func()
		var loadCurrentSong func(startAtEnd bool)
		var renderPage func()

		currentSongIdx := 0
		currentPage := 0
		totalPages := 0
		pagesToShow := 1

		var currentPdfMgr *pdf.Manager
		var viewerSize fyne.Size
		var activeTimerStopChan chan struct{}

		metroAudio, _ := audio.NewMetronomeAudio()
		metroIndicator := canvas.NewRectangle(theme.DisabledColor())
		metroIndicator.SetMinSize(fyne.NewSize(20, 20))
		metroIndicatorContainer := container.NewCenter(metroIndicator)

		var dialogBeatCb func(bool)
		if metroAudio != nil {
			metroAudio.OnBeat = func(isAccent bool) {
				if isAccent {
					metroIndicator.FillColor = theme.SuccessColor()
				} else {
					metroIndicator.FillColor = theme.PrimaryColor()
				}
				metroIndicator.Refresh()

				time.AfterFunc(100*time.Millisecond, func() {
					metroIndicator.FillColor = theme.DisabledColor()
					metroIndicator.Refresh()
				})
				if dialogBeatCb != nil {
					dialogBeatCb(isAccent)
				}

				if syncMode == 2 && isConnected && !previewMode {
					if syncStream != nil {
						syncStream.Send(&syncpb.SyncRequest{
							ConcertId: concert.ID,
							Action:    syncpb.ActionType_METRONOME_TICK,
							IsAccent:  isAccent,
							IsLeader:  true,
						})
					}
				}
			}
		}

		syncStatusBtn := widget.NewButton("Connecting...", nil)
		joinBtn := widget.NewButtonWithIcon("Join", theme.LoginIcon(), nil)
		leadBtn := widget.NewButtonWithIcon("Lead", theme.DocumentCreateIcon(), nil)
		stopLeadBtn := widget.NewButtonWithIcon("Stop Leading", theme.CancelIcon(), nil)
		leaveBtn := widget.NewButtonWithIcon("Leave Sync", theme.CancelIcon(), nil)
		previewBtn := widget.NewButtonWithIcon("Preview", theme.VisibilityIcon(), nil)
		pushBtn := widget.NewButtonWithIcon("Push", theme.UploadIcon(), nil)
		cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), nil)

		sendStateUpdate = func() {
			if !isConnected || syncMode != 2 || previewMode {
				return
			}
			if syncStream != nil {
				syncStream.Send(&syncpb.SyncRequest{
					ConcertId:    concert.ID,
					Action:       syncpb.ActionType_STATE_UPDATE,
					PageNumber:   uint32(currentPage),
					ItemIndex:    uint32(currentSongIdx),
					TimerSeconds: uint32(remainingSec),
					IsLeader:     true,
				})
			}
		}

		updateSyncUI = func() {
			syncStatusBtn.Hide()
			joinBtn.Hide()
			leadBtn.Hide()
			stopLeadBtn.Hide()
			leaveBtn.Hide()
			previewBtn.Hide()
			pushBtn.Hide()
			cancelBtn.Hide()

			if !isConnected {
				syncStatusBtn.SetText("Offline")
				syncStatusBtn.Show()
				return
			}

			syncStatusBtn.Show()

			if syncMode == 0 {
				leadBtn.Show()
				if leaderAvailable {
					joinBtn.Show()
					syncStatusBtn.SetText("Leader Active")
				} else {
					syncStatusBtn.SetText("Standby")
				}
			} else if syncMode == 1 {
				leaveBtn.Show()
				if autoFollow {
					syncStatusBtn.SetText("Following Leader")
				} else {
					diffStr := ""
					if currentSongIdx != leaderItemIdx {
						diffStr = fmt.Sprintf("Leader on Item %d", leaderItemIdx+1)
					} else {
						diff := currentPage - leaderPage
						if diff > 0 {
							diffStr = fmt.Sprintf("+%d Pages", diff)
						} else if diff < 0 {
							diffStr = fmt.Sprintf("%d Pages", diff)
						} else {
							diffStr = "Synced"
						}
					}
					syncStatusBtn.SetText(fmt.Sprintf("Diff: %s (Click to Sync)", diffStr))
				}
			} else if syncMode == 2 {
				stopLeadBtn.Show()
				if previewMode {
					syncStatusBtn.SetText("PREVIEW MODE")
					pushBtn.Show()
					cancelBtn.Show()
				} else {
					syncStatusBtn.SetText("LEADING")
					previewBtn.Show()
				}
			}
		}

		startSyncBackground = func() {
			token := app.Preferences().String("jwt_token")
			server := app.Preferences().String("server_addr")
			if token == "" || server == "" {
				isConnected = false
				updateSyncUI()
				return
			}
			conn, err := network.NewGRPCClient(server, token)
			if err != nil {
				isConnected = false
				updateSyncUI()
				return
			}
			syncConn = conn
			syncClient := syncpb.NewLiveSyncServiceClient(conn)
			syncCtx, syncCancel = context.WithCancel(context.Background())

			stream, err := syncClient.SyncConcertStream(syncCtx)
			if err != nil {
				syncCancel()
				syncConn.Close()
				isConnected = false
				updateSyncUI()
				return
			}

			syncStream = stream
			isConnected = true

			syncStream.Send(&syncpb.SyncRequest{
				ConcertId: concert.ID,
				Action:    syncpb.ActionType_UNKNOWN_ACTION,
				IsLeader:  false,
			})

			updateSyncUI()

			for {
				msg, err := syncStream.Recv()
				if err != nil {
					isConnected = false
					syncMode = 0
					leaderAvailable = false
					updateSyncUI()
					break
				}

				if msg.GetAction() == syncpb.ActionType_STOP_LEADING {
					leaderAvailable = false
					if syncMode == 1 {
						syncMode = 0
						autoFollow = false
					}
					updateSyncUI()
					continue
				}

				if msg.GetIsLeader() {
					leaderAvailable = true

					if msg.GetAction() == syncpb.ActionType_METRONOME_TICK {
						metroIndicator.FillColor = theme.SuccessColor()
						if !msg.GetIsAccent() {
							metroIndicator.FillColor = theme.PrimaryColor()
						}
						metroIndicator.Refresh()
						time.AfterFunc(100*time.Millisecond, func() {
							metroIndicator.FillColor = theme.DisabledColor()
							metroIndicator.Refresh()
						})
						if dialogBeatCb != nil {
							dialogBeatCb(msg.GetIsAccent())
						}
						continue
					}

					if msg.GetAction() == syncpb.ActionType_STATE_UPDATE {
						if syncMode == 2 {
							continue
						}

						leaderItemIdx = int(msg.GetItemIndex())
						leaderPage = int(msg.GetPageNumber())
						leaderTimer = int(msg.GetTimerSeconds())

						if syncMode == 1 && autoFollow {
							needLoad := currentSongIdx != leaderItemIdx
							currentSongIdx = leaderItemIdx
							currentPage = leaderPage
							remainingSec = leaderTimer

							if needLoad {
								loadCurrentSong(false)
							} else {
								renderPage()
							}

							if timerClockLabel != nil {
								isTimerRunning = false
								if startPauseBtn != nil {
									startPauseBtn.SetText("Start")
									startPauseBtn.SetIcon(theme.MediaPlayIcon())
								}
								timerClockLabel.Text = fmt.Sprintf("%02d:%02d", remainingSec/60, remainingSec%60)
								timerClockLabel.Refresh()
							}
						}
						updateSyncUI()
					}
				}
			}
		}

		leadBtn.OnTapped = func() {
			syncMode = 2
			previewMode = false
			sendStateUpdate()
			updateSyncUI()
		}

		stopLeadBtn.OnTapped = func() {
			if syncStream != nil {
				syncStream.Send(&syncpb.SyncRequest{
					ConcertId: concert.ID,
					Action:    syncpb.ActionType_STOP_LEADING,
				})
			}
			syncMode = 0
			leaderAvailable = false
			updateSyncUI()
		}

		joinBtn.OnTapped = func() {
			syncMode = 1
			autoFollow = true

			needLoad := currentSongIdx != leaderItemIdx
			currentSongIdx = leaderItemIdx
			currentPage = leaderPage
			remainingSec = leaderTimer

			if needLoad {
				loadCurrentSong(false)
			} else {
				renderPage()
			}

			if timerClockLabel != nil {
				isTimerRunning = false
				if startPauseBtn != nil {
					startPauseBtn.SetText("Start")
					startPauseBtn.SetIcon(theme.MediaPlayIcon())
				}
				timerClockLabel.Text = fmt.Sprintf("%02d:%02d", remainingSec/60, remainingSec%60)
				timerClockLabel.Refresh()
			}
			updateSyncUI()
		}

		leaveBtn.OnTapped = func() {
			syncMode = 0
			autoFollow = false
			updateSyncUI()
		}

		previewBtn.OnTapped = func() {
			previewMode = true
			savedItemIdx = currentSongIdx
			savedPage = currentPage
			savedTimer = remainingSec
			updateSyncUI()
		}

		pushBtn.OnTapped = func() {
			previewMode = false
			sendStateUpdate()
			updateSyncUI()
		}

		cancelBtn.OnTapped = func() {
			previewMode = false
			currentSongIdx = savedItemIdx
			currentPage = savedPage
			remainingSec = savedTimer
			loadCurrentSong(false)
			updateSyncUI()
		}

		syncStatusBtn.OnTapped = func() {
			if syncMode == 1 && !autoFollow {
				autoFollow = true
				currentSongIdx = leaderItemIdx
				currentPage = leaderPage
				remainingSec = leaderTimer
				loadCurrentSong(false)
				renderPage()

				if timerClockLabel != nil {
					isTimerRunning = false
					if startPauseBtn != nil {
						startPauseBtn.SetText("Start")
						startPauseBtn.SetIcon(theme.MediaPlayIcon())
					}
					timerClockLabel.Text = fmt.Sprintf("%02d:%02d", remainingSec/60, remainingSec%60)
					timerClockLabel.Refresh()
				}
				updateSyncUI()
			}
		}

		topSyncControls := container.NewHBox(
			syncStatusBtn,
			joinBtn,
			leadBtn,
			stopLeadBtn,
			leaveBtn,
			previewBtn,
			pushBtn,
			cancelBtn,
		)

		syncMode = 0
		leaderAvailable = false
		go startSyncBackground()

		concertClockLabel := canvas.NewText("--:--:--", theme.ForegroundColor())
		concertClockLabel.Alignment = fyne.TextAlignCenter
		concertClockLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		concertClockLabel.TextSize = 16

		parseStartTime := func(s string) (time.Time, bool) {
			s = strings.TrimSpace(s)
			if s == "" {
				return time.Time{}, false
			}

			now := time.Now()
			formats := []string{
				"2006-01-02 15:04:05",
				"2006-01-02 15:04",
				"2006-01-02T15:04:05Z07:00",
			}
			for _, f := range formats {
				if t, err := time.Parse(f, s); err == nil {
					return t, true
				}
			}
			timeFormats := []string{"15:04:05", "15:04"}
			for _, f := range timeFormats {
				if t, err := time.Parse(f, s); err == nil {
					tToday := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location())
					return tToday, true
				}
			}
			return time.Time{}, false
		}

		startTime, hasValidStartTime := parseStartTime(concert.StartTime)
		fallbackStartTime := time.Now()

		stopClockChan := make(chan struct{})
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopClockChan:
					return
				case <-ticker.C:
					now := time.Now()
					var diff time.Duration
					var prefix string

					if hasValidStartTime {
						if now.Before(startTime) {
							prefix = "-"
							diff = startTime.Sub(now)
						} else {
							prefix = ""
							diff = now.Sub(startTime)
						}
					} else {
						prefix = ""
						diff = now.Sub(fallbackStartTime)
					}

					totalSec := int(diff.Seconds())
					h := totalSec / 3600
					m := (totalSec % 3600) / 60
					s := totalSec % 60

					concertClockLabel.Text = fmt.Sprintf("%s%02d:%02d:%02d", prefix, h, m, s)
					concertClockLabel.Refresh()
				}
			}
		}()

		stopCurrentTimer := func() {
			if activeTimerStopChan != nil {
				close(activeTimerStopChan)
				activeTimerStopChan = nil
			}
		}

		pdfContainer := container.NewMax()
		songTitleLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		pageLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		renderPage = func() {
			if currentPdfMgr == nil || totalPages == 0 {
				return
			}
			pagesToShow = 1
			if viewerSize.Width > viewerSize.Height && viewerSize.Width > 0 && currentPage+1 < totalPages {
				pagesToShow = 2
			}

			if pagesToShow == 2 {
				img1, err1 := currentPdfMgr.GetPageImage(currentPage)
				img2, err2 := currentPdfMgr.GetPageImage(currentPage + 1)
				if err1 == nil && err2 == nil {
					canvasImg1 := canvas.NewImageFromImage(img1)
					canvasImg1.FillMode = canvas.ImageFillContain
					canvasImg2 := canvas.NewImageFromImage(img2)
					canvasImg2.FillMode = canvas.ImageFillContain

					grid := container.NewGridWithColumns(2, canvasImg1, canvasImg2)
					pdfContainer.Objects = []fyne.CanvasObject{grid}
					pdfContainer.Refresh()
					pageLabel.SetText(fmt.Sprintf("Pages %d-%d / %d", currentPage+1, currentPage+2, totalPages))
					return
				}
			}

			img, err := currentPdfMgr.GetPageImage(currentPage)
			if err != nil {
				pdfContainer.Objects = []fyne.CanvasObject{
					widget.NewLabelWithStyle(fmt.Sprintf("Error rendering page %d", currentPage+1), fyne.TextAlignCenter, fyne.TextStyle{}),
				}
				pdfContainer.Refresh()
				return
			}

			canvasImg := canvas.NewImageFromImage(img)
			canvasImg.FillMode = canvas.ImageFillContain
			pdfContainer.Objects = []fyne.CanvasObject{canvasImg}
			pdfContainer.Refresh()
			pageLabel.SetText(fmt.Sprintf("Page %d / %d", currentPage+1, totalPages))
		}

		formatTimerText := func(sec int) string {
			if sec < 0 {
				sec = 0
			}
			return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
		}

		loadCurrentSong = func(startAtEnd bool) {
			stopCurrentTimer()

			if metroAudio != nil {
				metroAudio.Stop()
			}

			if currentPdfMgr != nil {
				currentPdfMgr.Close()
				currentPdfMgr = nil
			}
			if currentSongIdx < 0 || currentSongIdx >= len(concert.Items) {
				return
			}

			item := concert.Items[currentSongIdx]

			if item.BreakMin != nil {
				breakDuration := *item.BreakMin
				songTitleLabel.SetText(fmt.Sprintf("%d/%d: Break (%d min)", currentSongIdx+1, len(concert.Items), breakDuration))
				pageLabel.SetText("Break")

				if !isTimerRunning && (syncMode != 1 || !autoFollow || syncMode == 2) {
					remainingSec = breakDuration * 60
				}
				isTimerRunning = false

				timerClockLabel = canvas.NewText(formatTimerText(remainingSec), theme.ForegroundColor())
				timerClockLabel.Alignment = fyne.TextAlignCenter
				timerClockLabel.TextStyle = fyne.TextStyle{Bold: true}
				timerClockLabel.TextSize = 48

				timerStatusLabel = widget.NewLabelWithStyle("PAUSED", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

				startPauseBtn = widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), func() {
					if syncMode == 1 {
						autoFollow = false
						updateSyncUI()
					}
					if isTimerRunning {
						isTimerRunning = false
						stopCurrentTimer()
						startPauseBtn.SetText("Start")
						startPauseBtn.SetIcon(theme.MediaPlayIcon())
						timerStatusLabel.SetText("PAUSED")
						sendStateUpdate()
					} else {
						if remainingSec <= 0 {
							return
						}
						isTimerRunning = true
						startPauseBtn.SetText("Pause")
						startPauseBtn.SetIcon(theme.MediaPauseIcon())
						timerStatusLabel.SetText("COUNTDOWN IN PROGRESS")
						sendStateUpdate()

						stopCh := make(chan struct{})
						activeTimerStopChan = stopCh

						go func(stop <-chan struct{}) {
							ticker := time.NewTicker(time.Second)
							defer ticker.Stop()
							for {
								select {
								case <-stop:
									return
								case <-ticker.C:
									if isTimerRunning {
										remainingSec--
										if timerClockLabel != nil {
											timerClockLabel.Text = formatTimerText(remainingSec)
											timerClockLabel.Refresh()
										}
										if remainingSec <= 0 {
											isTimerRunning = false
											if timerStatusLabel != nil {
												timerStatusLabel.SetText("BREAK FINISHED!")
											}
											if startPauseBtn != nil {
												startPauseBtn.SetText("Start")
												startPauseBtn.SetIcon(theme.MediaPlayIcon())
											}
										}
										if syncMode == 2 && isConnected && !previewMode {
											sendStateUpdate()
										}
									}
								}
							}
						}(stopCh)
					}
				})
				startPauseBtn.Importance = widget.HighImportance

				resetBtn := widget.NewButtonWithIcon("Reset", theme.ViewRefreshIcon(), func() {
					if syncMode == 1 {
						autoFollow = false
						updateSyncUI()
					}
					isTimerRunning = false
					stopCurrentTimer()
					remainingSec = breakDuration * 60
					timerClockLabel.Text = formatTimerText(remainingSec)
					timerClockLabel.Refresh()
					timerStatusLabel.SetText("PAUSED")
					startPauseBtn.SetText("Start")
					startPauseBtn.SetIcon(theme.MediaPlayIcon())
					sendStateUpdate()
				})

				addMinBtn := widget.NewButton("+1 Min", func() {
					if syncMode == 1 {
						autoFollow = false
						updateSyncUI()
					}
					remainingSec += 60
					timerClockLabel.Text = formatTimerText(remainingSec)
					timerClockLabel.Refresh()
					sendStateUpdate()
				})
				subMinBtn := widget.NewButton("-1 Min", func() {
					if syncMode == 1 {
						autoFollow = false
						updateSyncUI()
					}
					if remainingSec > 60 {
						remainingSec -= 60
					} else {
						remainingSec = 0
					}
					timerClockLabel.Text = formatTimerText(remainingSec)
					timerClockLabel.Refresh()
					sendStateUpdate()
				})

				timerControls := container.NewHBox(
					layout.NewSpacer(),
					startPauseBtn, resetBtn, subMinBtn, addMinBtn,
					layout.NewSpacer(),
				)

				breakView := container.NewVBox(
					layout.NewSpacer(),
					widget.NewLabelWithStyle("STAGE BREAK", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
					timerClockLabel,
					timerStatusLabel,
					widget.NewLabel(""),
					timerControls,
					layout.NewSpacer(),
				)

				pdfContainer.Objects = []fyne.CanvasObject{container.NewCenter(breakView)}
				pdfContainer.Refresh()
				totalPages = 0
				return
			}

			title := "Unknown Item"
			if item.ScoreName != nil {
				title = *item.ScoreName
			}

			songTitleLabel.SetText(fmt.Sprintf("%d/%d: %s", currentSongIdx+1, len(concert.Items), title))

			if item.ScoreID != nil && item.FilePath != nil && *item.FilePath != "" {
				pdfMgr, err := pdf.NewManager(*item.FilePath)
				if err != nil {
					pdfContainer.Objects = []fyne.CanvasObject{
						widget.NewLabelWithStyle(fmt.Sprintf("PDF file not found:\n%s", *item.FilePath), fyne.TextAlignCenter, fyne.TextStyle{}),
					}
					pdfContainer.Refresh()
					totalPages = 0
					pageLabel.SetText("Page 0/0")
					return
				}
				currentPdfMgr = pdfMgr
				totalPages = pdfMgr.GetPageCount()
				if startAtEnd && totalPages > 0 {
					currentPage = totalPages - 1
					if viewerSize.Width > viewerSize.Height && totalPages >= 2 {
						currentPage = totalPages - 2
					}
				} else {
					currentPage = 0
				}
				renderPage()
			} else {
				pdfContainer.Objects = []fyne.CanvasObject{
					widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				}
				pdfContainer.Refresh()
				totalPages = 0
				pageLabel.SetText("No File")
			}
		}

		exitConcertBtn := widget.NewButtonWithIcon("Exit", theme.CancelIcon(), func() {
			if syncMode == 2 && syncStream != nil {
				syncStream.Send(&syncpb.SyncRequest{
					ConcertId: concert.ID,
					Action:    syncpb.ActionType_STOP_LEADING,
				})
			}
			stopCurrentTimer()
			close(stopClockChan)
			if metroAudio != nil {
				metroAudio.Stop()
				metroAudio.Close()
			}
			if currentPdfMgr != nil {
				currentPdfMgr.Close()
			}
			if isConnected {
				syncCancel()
				syncConn.Close()
			}
			showConcertList()
		})
		exitConcertBtn.Importance = widget.DangerImportance

		prevSongBtn := widget.NewButtonWithIcon("Prev Item", theme.MediaSkipPreviousIcon(), func() {
			if syncMode == 1 {
				autoFollow = false
			}
			if currentSongIdx > 0 {
				currentSongIdx--
				loadCurrentSong(false)
			}
			sendStateUpdate()
			updateSyncUI()
		})
		nextSongBtn := widget.NewButtonWithIcon("Next Item", theme.MediaSkipNextIcon(), func() {
			if syncMode == 1 {
				autoFollow = false
			}
			if currentSongIdx < len(concert.Items)-1 {
				currentSongIdx++
				loadCurrentSong(false)
			}
			sendStateUpdate()
			updateSyncUI()
		})

		var setlistDialog dialog.Dialog
		setlistBtn := widget.NewButtonWithIcon("Setlist", theme.ListIcon(), func() {
			var items []fyne.CanvasObject
			for i, item := range concert.Items {
				idx := i
				var title string
				if item.ScoreName != nil {
					title = *item.ScoreName
				} else if item.BreakMin != nil {
					title = fmt.Sprintf("Break (%d min)", *item.BreakMin)
				} else {
					title = "Unknown Item"
				}

				btn := widget.NewButton(fmt.Sprintf("%d. %s", idx+1, title), func() {
					if syncMode == 1 {
						autoFollow = false
					}
					currentSongIdx = idx
					loadCurrentSong(false)
					sendStateUpdate()
					updateSyncUI()
					if setlistDialog != nil {
						setlistDialog.Hide()
					}
				})
				if idx == currentSongIdx {
					btn.Importance = widget.HighImportance
					btn.SetIcon(theme.MediaPlayIcon())
				}
				items = append(items, btn)
			}
			scroll := container.NewVScroll(container.NewVBox(items...))
			scroll.SetMinSize(fyne.NewSize(350, 400))
			setlistDialog = dialog.NewCustom("Concert Setlist", "Close", scroll, w)
			setlistDialog.Show()
		})
		setlistBtn.Importance = widget.HighImportance

		toolsBtn := widget.NewButtonWithIcon("Tools", theme.SettingsIcon(), func() {
			ShowToolsMenu(w, metroAudio, func(cb func(bool)) {
				dialogBeatCb = cb
			})
		})

		prevPageBtn := widget.NewButtonWithIcon("PREV\nPAGE", theme.NavigateBackIcon(), func() {
			if syncMode == 1 {
				autoFollow = false
			}
			if currentPage > 0 {
				currentPage -= pagesToShow
				if currentPage < 0 {
					currentPage = 0
				}
				renderPage()
			} else if currentSongIdx > 0 {
				currentSongIdx--
				loadCurrentSong(true)
			}
			sendStateUpdate()
			updateSyncUI()
		})
		prevPageBtn.Importance = widget.HighImportance

		nextPageBtn := widget.NewButtonWithIcon("NEXT\nPAGE", theme.NavigateNextIcon(), func() {
			if syncMode == 1 {
				autoFollow = false
			}
			if currentPage+pagesToShow < totalPages {
				currentPage += pagesToShow
				renderPage()
			} else if currentSongIdx < len(concert.Items)-1 {
				currentSongIdx++
				loadCurrentSong(false)
			}
			sendStateUpdate()
			updateSyncUI()
		})
		nextPageBtn.Importance = widget.HighImportance

		rightSidebar := container.NewBorder(
			container.NewVBox(pageLabel, widget.NewSeparator(), setlistBtn, toolsBtn, widget.NewSeparator()),
			nil, nil, nil,
			container.NewGridWithRows(2, prevPageBtn, nextPageBtn),
		)

		topRightControls := container.NewHBox(
			topSyncControls,
			widget.NewSeparator(),
			metroIndicatorContainer,
			widget.NewLabel(" "),
			concertClockLabel,
			widget.NewLabel(" "),
			prevSongBtn,
			nextSongBtn,
		)
		topBar := container.NewBorder(nil, nil, exitConcertBtn, topRightControls, songTitleLabel)

		viewer := NewResponsiveViewer(func(size fyne.Size) {
			viewerSize = size
			renderPage()
		})
		viewer.Content.Objects = []fyne.CanvasObject{pdfContainer}

		mainView := container.NewBorder(
			container.NewPadded(topBar),
			nil, nil,
			container.NewPadded(rightSidebar),
			viewer,
		)

		contentWrapper.Objects = []fyne.CanvasObject{mainView}
		contentWrapper.Refresh()

		loadCurrentSong(false)
	}

	showConcertList()
	return contentWrapper
}
