package ui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
	"github.com/ziomciopoziomcio/digital-music-stand/client/remote"
	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
)

func BuildConcertMode(w fyne.Window, app fyne.App, db *localdb.DBManager, remoteServer *remote.Server, goBack func(), openSetup func(editingConcert *localdb.Concert), onDeleteConcert func(), forceSync func(), showLockScreen func(), verifyPin func(string) bool, prefToken string, prefServer string, profilePath string) *fyne.Container {
	contentWrapper := container.NewMax()

	var showConcertList func()
	var updateGrid func()
	var playConcert func(concert localdb.Concert)

	gridWrapper := container.NewMax()
	searchEntry := NewAutoKeyboardEntry()
	searchEntry.SetPlaceHolder("Search concerts (min. 3 chars)...")

	updateGrid = func() {
		isLoggedIn := app.Preferences().String(prefToken) != ""
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

			openBtn := NewTouchButtonWithIcon("ENTER", theme.MediaPlayIcon(), func() {
				playConcert(concert)
			})
			openBtn.Importance = widget.HighImportance

			var actionButtons *fyne.Container

			if concert.IsOwner {
				shareBtn := NewTouchButtonWithIcon("", theme.MailSendIcon(), func() {
					ShowAccessDialog(w, app, "Share Concert", concert.Name, "Share", true, func(email *string, bandID *uint32, canEdit bool) error {
						token := app.Preferences().String(prefToken)
						server := app.Preferences().String(prefServer)
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

				revokeBtn := NewTouchButtonWithIcon("", theme.ContentRemoveIcon(), func() {
					ShowAccessDialog(w, app, "Revoke Concert Access", concert.Name, "Revoke", false, func(email *string, bandID *uint32, canEdit bool) error {
						token := app.Preferences().String(prefToken)
						server := app.Preferences().String(prefServer)
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

				editBtn := NewTouchButtonWithIcon("", theme.DocumentCreateIcon(), func() {
					openSetup(&concert)
				})

				deleteBtn := NewTouchButtonWithIcon("", theme.DeleteIcon(), func() {
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

				setAliasBtn := NewTouchButtonWithIcon("Alias", theme.SettingsIcon(), func() {
					entry := NewAutoKeyboardEntry()
					entry.SetText(concert.DisplayName())
					var d dialog.Dialog
					formContent := container.NewVBox(
						widget.NewLabel("Set local alias (only visible to you):"),
						entry,
						container.NewHBox(
							layout.NewSpacer(),
							NewTouchButton("Save", func() {
								_ = db.SetConcertAlias(concert.ID, entry.Text)
								updateGrid()
								d.Hide()
							}),
							NewTouchButton("Clear", func() {
								_ = db.SetConcertAlias(concert.ID, "")
								updateGrid()
								d.Hide()
							}),
							NewTouchButton("Cancel", func() { d.Hide() }),
						),
					)
					d = dialog.NewCustomWithoutButtons("Set Alias", formContent, w)
					d.Show()
					w.Canvas().Focus(entry)
				})

				editBtn := NewTouchButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
					openSetup(&concert)
				})

				actionButtons = container.NewHBox(sharedBadge, setAliasBtn, editBtn, openBtn)
			} else {
				sharedBadge := widget.NewLabelWithStyle("Shared", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
				setAliasBtn := NewTouchButtonWithIcon("Alias", theme.SettingsIcon(), func() {
					entry := NewAutoKeyboardEntry()
					entry.SetText(concert.DisplayName())
					var d dialog.Dialog
					formContent := container.NewVBox(
						widget.NewLabel("Set local alias (only visible to you):"),
						entry,
						container.NewHBox(
							layout.NewSpacer(),
							NewTouchButton("Save", func() {
								_ = db.SetConcertAlias(concert.ID, entry.Text)
								updateGrid()
								d.Hide()
							}),
							NewTouchButton("Clear", func() {
								_ = db.SetConcertAlias(concert.ID, "")
								updateGrid()
								d.Hide()
							}),
							NewTouchButton("Cancel", func() { d.Hide() }),
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
		backBtn := NewTouchButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backBtn.Importance = widget.WarningImportance

		newConcertBtn := NewTouchButtonWithIcon("New Concert", theme.ContentAddIcon(), func() {
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

		var exitConcertBtn, prevSongBtn, nextSongBtn, prevPageBtn, nextPageBtn *widget.Button
		var handleRemoteCommand func(action string)
		var stopGaze func()

		var cloudConn *grpc.ClientConn
		var p2pConn *grpc.ClientConn
		var cloudCancel context.CancelFunc
		var p2pCancel context.CancelFunc
		var syncStreamCloud syncpb.LiveSyncService_SyncConcertStreamClient

		var isConnectedCloud bool
		var isConnectedP2P bool
		var wantsToSync bool

		var isLeader bool
		var previewMode bool
		var autoFollow bool
		var isLocked bool

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

		var sendStateUpdate func()
		var updateSyncUI func()
		var loadCurrentSong func(startAtEnd bool)
		var renderPage func()
		var updateRemoteState func()

		var stopClockOnce sync.Once
		stopClockChan := make(chan struct{})

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

		recorderAudio, _ := audio.NewRecorderAudio()
		recIndicator := canvas.NewRectangle(color.Transparent)
		recIndicator.SetMinSize(fyne.NewSize(20, 20))
		recIndicator.CornerRadius = 10
		recIndicatorContainer := container.NewCenter(recIndicator)

		if recorderAudio != nil {
			recorderAudio.OnRecordPulse = func(active bool) {
				if active {
					recIndicator.FillColor = theme.ErrorColor()
				} else {
					recIndicator.FillColor = color.Transparent
				}
				recIndicator.Refresh()
			}
		}

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

				if isLeader && !previewMode {
					req := &syncpb.SyncRequest{
						ConcertId: concert.ID,
						Action:    syncpb.ActionType_METRONOME_TICK,
						IsAccent:  isAccent,
						IsLeader:  true,
					}
					if isConnectedCloud && syncStreamCloud != nil {
						_ = syncStreamCloud.Send(req)
					}
					remoteServer.BroadcastLocal(req)
				}
			}
		}

		syncStatusBtn := NewTouchButton("Offline", nil)
		joinBtn := NewTouchButtonWithIcon("Join", theme.LoginIcon(), nil)
		leadBtn := NewTouchButtonWithIcon("Lead", theme.DocumentCreateIcon(), nil)
		previewBtn := NewTouchButtonWithIcon("Preview", theme.VisibilityIcon(), nil)
		pushBtn := NewTouchButtonWithIcon("Push", theme.UploadIcon(), nil)
		cancelBtn := NewTouchButtonWithIcon("Cancel", theme.CancelIcon(), nil)
		exitSyncBtn := NewTouchButtonWithIcon("Exit Sync", theme.CancelIcon(), nil)

		sendStateUpdate = func() {
			if !isLeader || previewMode {
				return
			}

			req := &syncpb.SyncRequest{
				ConcertId:    concert.ID,
				Action:       syncpb.ActionType_STATE_UPDATE,
				PageNumber:   uint32(currentPage),
				ItemIndex:    uint32(currentSongIdx),
				TimerSeconds: uint32(remainingSec),
				IsLeader:     isLeader,
			}

			if isConnectedCloud && syncStreamCloud != nil {
				_ = syncStreamCloud.Send(req)
			}
			remoteServer.BroadcastLocal(req)
		}

		var setlistTitles []string
		for _, it := range concert.Items {
			if it.ScoreName != nil {
				setlistTitles = append(setlistTitles, *it.ScoreName)
			} else if it.BreakMin != nil {
				setlistTitles = append(setlistTitles, fmt.Sprintf("Break (%d min)", *it.BreakMin))
			} else {
				setlistTitles = append(setlistTitles, "Unknown Item")
			}
		}

		updateRemoteState = func() {
			path := ""
			scoreID := ""
			if currentPdfMgr != nil && currentSongIdx < len(concert.Items) {
				if concert.Items[currentSongIdx].FilePath != nil {
					path = *concert.Items[currentSongIdx].FilePath
				}
				if concert.Items[currentSongIdx].ScoreID != nil {
					scoreID = *concert.Items[currentSongIdx].ScoreID
				}
			}
			remoteServer.SetState(remote.ConcertState{
				ConcertName:    concert.Name,
				CurrentItemIdx: currentSongIdx,
				CurrentPage:    currentPage,
				TotalPages:     totalPages,
				IsTimerRunning: isTimerRunning,
				TimerSeconds:   remainingSec,
				IsLocked:       isLocked,
				Setlist:        setlistTitles,
				CurrentScoreID: scoreID,
				CurrentPDFPath: path,
			})
		}

		updateSyncUI = func() {
			syncStatusBtn.Hide()
			joinBtn.Hide()
			leadBtn.Hide()
			previewBtn.Hide()
			pushBtn.Hide()
			cancelBtn.Hide()
			exitSyncBtn.Hide()

			if !wantsToSync {
				syncStatusBtn.SetText("Offline")
				syncStatusBtn.Show()
				joinBtn.Show()
				leadBtn.Show()
				return
			}

			syncStatusBtn.Show()
			exitSyncBtn.Show()

			connText := "Cloud"
			if isConnectedP2P {
				connText = "P2P LAN"
			}
			if !isConnectedCloud && !isConnectedP2P {
				connText = "Searching..."
			}

			if isLeader {
				if previewMode {
					syncStatusBtn.SetText(fmt.Sprintf("PREVIEW (%s)", connText))
					pushBtn.Show()
					cancelBtn.Show()
				} else {
					syncStatusBtn.SetText(fmt.Sprintf("LEADING (%s)", connText))
					previewBtn.Show()
				}
			} else {
				if autoFollow {
					syncStatusBtn.SetText(fmt.Sprintf("Following (%s)", connText))
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
			}
		}

		handleSyncMessage := func(msg *syncpb.SyncResponse) {
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
				return
			}

			if isLeader && !previewMode {
				if msg.GetAction() == syncpb.ActionType_UNKNOWN_ACTION {
					ip := webserver.GetLocalIP()
					pin := remoteServer.GetPIN()
					payload := fmt.Sprintf("%s|%s", ip, pin)

					if isConnectedCloud && syncStreamCloud != nil {
						_ = syncStreamCloud.Send(&syncpb.SyncRequest{
							ConcertId: concert.ID,
							Action:    syncpb.ActionType_BROADCAST_INFO,
							Payload:   payload,
							IsLeader:  true,
						})
					}
				}

				switch msg.GetAction() {
				case syncpb.ActionType_NEXT_PAGE:
					if handleRemoteCommand != nil {
						handleRemoteCommand("NEXT_PAGE")
					}
				case syncpb.ActionType_PREV_PAGE:
					if handleRemoteCommand != nil {
						handleRemoteCommand("PREV_PAGE")
					}
				case syncpb.ActionType_NEXT_ITEM:
					if handleRemoteCommand != nil {
						handleRemoteCommand("NEXT_ITEM")
					}
				case syncpb.ActionType_PREV_ITEM:
					if handleRemoteCommand != nil {
						handleRemoteCommand("PREV_ITEM")
					}
				case syncpb.ActionType_TOGGLE_TIMER:
					if handleRemoteCommand != nil {
						handleRemoteCommand("TOGGLE_TIMER")
					}
				}
				return
			}

			if msg.GetAction() == syncpb.ActionType_STATE_UPDATE && msg.GetIsLeader() {
				if isLeader {
					return
				}

				leaderItemIdx = int(msg.GetItemIndex())
				leaderPage = int(msg.GetPageNumber())
				leaderTimer = int(msg.GetTimerSeconds())

				if autoFollow {
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

		connectCloud := func() bool {
			token := app.Preferences().String(prefToken)
			server := app.Preferences().String(prefServer)
			if token == "" || server == "" {
				return false
			}

			conn, err := network.NewGRPCClient(server, token)
			if err != nil {
				return false
			}

			client := syncpb.NewLiveSyncServiceClient(conn)
			ctx, cancel := context.WithCancel(context.Background())
			stream, err := client.SyncConcertStream(ctx)
			if err != nil {
				cancel()
				conn.Close()
				return false
			}

			cloudConn = conn
			cloudCancel = cancel
			syncStreamCloud = stream
			isConnectedCloud = true

			_ = stream.Send(&syncpb.SyncRequest{
				ConcertId: concert.ID,
				Action:    syncpb.ActionType_UNKNOWN_ACTION,
				IsLeader:  false,
			})

			if isLeader {
				sendStateUpdate()
			}

			go func() {
				for {
					msg, err := stream.Recv()
					if err != nil {
						break
					}
					handleSyncMessage(msg)
				}
				isConnectedCloud = false
				updateSyncUI()
			}()
			return true
		}

		connectP2P := func(ip string, port int) bool {
			addr := fmt.Sprintf("%s:%d", ip, port)
			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return false
			}

			client := syncpb.NewLiveSyncServiceClient(conn)
			ctx, cancel := context.WithCancel(context.Background())
			stream, err := client.SyncConcertStream(ctx)
			if err != nil {
				cancel()
				conn.Close()
				return false
			}

			p2pConn = conn
			p2pCancel = cancel
			isConnectedP2P = true

			_ = stream.Send(&syncpb.SyncRequest{
				ConcertId: concert.ID,
				Action:    syncpb.ActionType_UNKNOWN_ACTION,
				IsLeader:  false,
			})

			go func() {
				for {
					msg, err := stream.Recv()
					if err != nil {
						break
					}
					if !isConnectedCloud {
						handleSyncMessage(msg)
					}
				}
				isConnectedP2P = false
				updateSyncUI()
			}()
			return true
		}

		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopClockChan:
					if isConnectedCloud {
						cloudCancel()
						cloudConn.Close()
					}
					if isConnectedP2P {
						p2pCancel()
						p2pConn.Close()
					}
					return
				case <-ticker.C:
					if !wantsToSync {
						continue
					}

					if !isConnectedCloud {
						if connectCloud() {
							if isConnectedP2P {
								p2pCancel()
								p2pConn.Close()
								isConnectedP2P = false
							}
							updateSyncUI()
						}
					}

					if !isConnectedCloud && !isLeader && !isConnectedP2P {
						peer, found := remoteServer.GetPeerForConcert(concert.ID)
						if found {
							if connectP2P(peer.IP, peer.Port) {
								updateSyncUI()
							}
						}
					}
				}
			}
		}()

		joinBtn.OnTapped = func() {
			wantsToSync = true
			isLeader = false
			autoFollow = true
			remoteServer.SetLeading("", false)
			updateSyncUI()
		}

		leadBtn.OnTapped = func() {
			wantsToSync = true
			isLeader = true
			autoFollow = false
			previewMode = false
			remoteServer.SetLeading(concert.ID, true)
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

		exitSyncBtn.OnTapped = func() {
			wantsToSync = false
			isLeader = false
			remoteServer.SetLeading("", false)
			if isConnectedCloud {
				cloudCancel()
				cloudConn.Close()
				isConnectedCloud = false
			}
			if isConnectedP2P {
				p2pCancel()
				p2pConn.Close()
				isConnectedP2P = false
			}
			updateSyncUI()
		}

		syncStatusBtn.OnTapped = func() {
			if !wantsToSync {
				return
			}
			if !isLeader && !autoFollow {
				autoFollow = true
				currentSongIdx = leaderItemIdx
				currentPage = leaderPage
				remainingSec = leaderTimer
				loadCurrentSong(false)
				renderPage()
				updateSyncUI()
			}
		}

		topSyncControls := container.NewHBox(
			syncStatusBtn,
			joinBtn,
			leadBtn,
			previewBtn,
			pushBtn,
			cancelBtn,
			exitSyncBtn,
		)

		updateSyncUI()

		lockOverlay := container.NewMax()
		lockBg := canvas.NewRectangle(color.Black)

		unlockBtn := NewTouchButtonWithIcon("Unlock Screen", theme.LoginIcon(), func() {
			pinEntry := widget.NewPasswordEntry()
			dialog.ShowCustomConfirm("Unlock Device", "Unlock", "Cancel", pinEntry, func(ok bool) {
				if ok {
					if verifyPin != nil && verifyPin(pinEntry.Text) {
						isLocked = false
						lockOverlay.Hide()
						updateRemoteState()
					} else {
						dialog.ShowInformation("Error", "Invalid Profile PIN", w)
					}
				}
			}, w)
		})
		unlockBtn.Importance = widget.HighImportance

		lockOverlay.Objects = []fyne.CanvasObject{
			lockBg,
			container.NewCenter(container.NewVBox(
				widget.NewLabelWithStyle("DEVICE LOCKED BY REMOTE", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				unlockBtn,
			)),
		}
		lockOverlay.Hide()

		handleRemoteCommand = func(action string) {
			if isLocked && action != "UNLOCK_SCREEN" && action != "TOGGLE_TIMER" {
				return
			}
			switch action {
			case "NEXT_PAGE":
				if nextPageBtn != nil {
					nextPageBtn.OnTapped()
				}
			case "PREV_PAGE":
				if prevPageBtn != nil {
					prevPageBtn.OnTapped()
				}
			case "NEXT_ITEM":
				if nextSongBtn != nil {
					nextSongBtn.OnTapped()
				}
			case "PREV_ITEM":
				if prevSongBtn != nil {
					prevSongBtn.OnTapped()
				}
			case "TOGGLE_TIMER":
				if startPauseBtn != nil {
					startPauseBtn.OnTapped()
				}
			case "LOCK_SCREEN":
				isLocked = true
				lockOverlay.Show()
				updateRemoteState()
			case "UNLOCK_SCREEN":
				isLocked = false
				lockOverlay.Hide()
				updateRemoteState()
			}
		}

		for len(remoteServer.CommandChan) > 0 {
			<-remoteServer.CommandChan
		}

		go func() {
			for {
				select {
				case <-stopClockChan:
					return
				case cmd := <-remoteServer.CommandChan:
					handleRemoteCommand(cmd.Action)
				}
			}
		}()

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
				}
			} else {
				img, err := currentPdfMgr.GetPageImage(currentPage)
				if err != nil {
					pdfContainer.Objects = []fyne.CanvasObject{
						widget.NewLabelWithStyle(fmt.Sprintf("Error rendering page %d", currentPage+1), fyne.TextAlignCenter, fyne.TextStyle{}),
					}
					pdfContainer.Refresh()
				} else {
					canvasImg := canvas.NewImageFromImage(img)
					canvasImg.FillMode = canvas.ImageFillContain
					pdfContainer.Objects = []fyne.CanvasObject{canvasImg}
					pdfContainer.Refresh()
					pageLabel.SetText(fmt.Sprintf("Page %d / %d", currentPage+1, totalPages))
				}
			}
			updateRemoteState()
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

				if !isTimerRunning && (!autoFollow || isLeader) {
					remainingSec = breakDuration * 60
				}
				isTimerRunning = false

				timerClockLabel = canvas.NewText(formatTimerText(remainingSec), theme.ForegroundColor())
				timerClockLabel.Alignment = fyne.TextAlignCenter
				timerClockLabel.TextStyle = fyne.TextStyle{Bold: true}
				timerClockLabel.TextSize = 48

				timerStatusLabel = widget.NewLabelWithStyle("PAUSED", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

				startPauseBtn = NewTouchButtonWithIcon("Start", theme.MediaPlayIcon(), func() {
					if !isLeader {
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
						updateRemoteState()
					} else {
						if remainingSec <= 0 {
							return
						}
						isTimerRunning = true
						startPauseBtn.SetText("Pause")
						startPauseBtn.SetIcon(theme.MediaPauseIcon())
						timerStatusLabel.SetText("COUNTDOWN IN PROGRESS")
						sendStateUpdate()
						updateRemoteState()

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
										updateRemoteState()

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
										if isLeader && !previewMode {
											sendStateUpdate()
										}
									}
								}
							}
						}(stopCh)
					}
				})
				startPauseBtn.Importance = widget.HighImportance

				resetBtn := NewTouchButtonWithIcon("Reset", theme.ViewRefreshIcon(), func() {
					if !isLeader {
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
					updateRemoteState()
				})

				addMinBtn := NewTouchButton("+1 Min", func() {
					if !isLeader {
						autoFollow = false
						updateSyncUI()
					}
					remainingSec += 60
					timerClockLabel.Text = formatTimerText(remainingSec)
					timerClockLabel.Refresh()
					sendStateUpdate()
					updateRemoteState()
				})
				subMinBtn := NewTouchButton("-1 Min", func() {
					if !isLeader {
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
					updateRemoteState()
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
				updateRemoteState()
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
					updateRemoteState()
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
				updateRemoteState()
			}
		}

		exitConcertBtn = NewTouchButtonWithIcon("Exit", theme.CancelIcon(), func() {
			if isLeader {
				remoteServer.SetLeading("", false)
			}
			wantsToSync = false

			if isLeader && syncStreamCloud != nil {
				_ = syncStreamCloud.Send(&syncpb.SyncRequest{
					ConcertId: concert.ID,
					Action:    syncpb.ActionType_STOP_LEADING,
				})
			}
			stopCurrentTimer()

			stopClockOnce.Do(func() {
				close(stopClockChan)
			})

			if metroAudio != nil {
				metroAudio.Stop()
				metroAudio.Close()
			}
			if currentPdfMgr != nil {
				currentPdfMgr.Close()
			}
			if recorderAudio != nil {
				recorderAudio.Close()
			}
			showConcertList()
			if stopGaze != nil {
				stopGaze()
			}
		})
		exitConcertBtn.Importance = widget.DangerImportance

		prevSongBtn = NewTouchButtonWithIcon("Prev Item", theme.MediaSkipPreviousIcon(), func() {
			if !isLeader {
				autoFollow = false
			}
			if currentSongIdx > 0 {
				currentSongIdx--
				loadCurrentSong(false)
			}
			sendStateUpdate()
			updateSyncUI()
		})
		nextSongBtn = NewTouchButtonWithIcon("Next Item", theme.MediaSkipNextIcon(), func() {
			if !isLeader {
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
		setlistBtn := NewTouchButtonWithIcon("Setlist", theme.ListIcon(), func() {
			var items []fyne.CanvasObject
			for i, title := range setlistTitles {
				idx := i
				btn := NewTouchButton(fmt.Sprintf("%d. %s", idx+1, title), func() {
					if !isLeader {
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
			setlistDialog = dialog.NewCustom("Concert Setlist", "Close", scroll, w)

			winSize := w.Canvas().Size()
			targetWidth := float32(350)
			targetHeight := float32(400)

			if winSize.Width < targetWidth {
				targetWidth = winSize.Width * 0.95
			}
			if winSize.Height < targetHeight {
				targetHeight = winSize.Height * 0.95
			}

			setlistDialog.Resize(fyne.NewSize(targetWidth, targetHeight))
			setlistDialog.Show()
		})
		setlistBtn.Importance = widget.HighImportance

		toolsBtn := NewTouchButtonWithIcon("Tools", theme.SettingsIcon(), func() {
			var currentScoreID string
			var currentScoreTitle string

			if currentSongIdx >= 0 && currentSongIdx < len(concert.Items) {
				item := concert.Items[currentSongIdx]
				if item.ScoreID != nil {
					currentScoreID = *item.ScoreID
				}
				if item.ScoreName != nil {
					currentScoreTitle = *item.ScoreName
				} else if item.BreakMin != nil {
					currentScoreTitle = fmt.Sprintf("Break (%d min)", *item.BreakMin)
				} else {
					currentScoreTitle = "Unknown Item"
				}
			}

			ShowToolsMenu(w, app, metroAudio, recorderAudio, remoteServer, db, currentScoreID, currentScoreTitle, profilePath, func(cb func(bool)) {
				dialogBeatCb = cb
			})
		})

		prevPageBtn = NewTouchButtonWithIcon("PREV\nPAGE", theme.NavigateBackIcon(), func() {
			if !isLeader {
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

		nextPageBtn = NewTouchButtonWithIcon("NEXT\nPAGE", theme.NavigateNextIcon(), func() {
			if !isLeader {
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
			topSyncControls, widget.NewSeparator(), metroIndicatorContainer,
			widget.NewLabel(" "), recIndicatorContainer, widget.NewLabel(" "),
			concertClockLabel, widget.NewLabel(" "),
			prevSongBtn, nextSongBtn,
		)
		topBar := container.NewBorder(nil, nil, exitConcertBtn, topRightControls, songTitleLabel)

		viewer := NewResponsiveViewer(func(size fyne.Size) {
			viewerSize = size
			renderPage()
		})
		viewer.Content.Objects = []fyne.CanvasObject{pdfContainer}

		overlay, stopper := NewGazeOverlay(w, app, func() {
			if prevPageBtn != nil && !prevPageBtn.Disabled() {
				prevPageBtn.OnTapped()
			}
		}, func() {
			if nextPageBtn != nil && !nextPageBtn.Disabled() {
				nextPageBtn.OnTapped()
			}
		}, func() bool { return isLocked })
		stopGaze = stopper

		mainView := container.NewMax(
			container.NewBorder(container.NewPadded(topBar), nil, nil, container.NewPadded(rightSidebar), viewer),
			overlay,
			lockOverlay,
		)

		contentWrapper.Objects = []fyne.CanvasObject{mainView}
		contentWrapper.Refresh()

		loadCurrentSong(false)
	}

	showConcertList()
	return contentWrapper
}
