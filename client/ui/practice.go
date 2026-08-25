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
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
)

func BuildPracticeMode(w fyne.Window, app fyne.App, db *localdb.DBManager, onScoresChanged func(), goBack func()) *fyne.Container {
	contentWrapper := container.NewMax()
	editMode := false

	var showLibrary func()
	var updateGrid func()

	gridWrapper := container.NewMax()
	searchEntry := NewAutoKeyboardEntry()
	searchEntry.SetPlaceHolder("Search scores (min. 3 chars)...")

	showScore := func(score localdb.Score) {
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
			}
		}

		pdfMgr, err := pdf.NewManager(score.FilePath)
		currentPage := 0
		totalPages := 0

		if err == nil {
			totalPages = pdfMgr.GetPageCount()
		}

		pageLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		currentPagesToShow := 1

		getPagesToShow := func(size fyne.Size) int {
			if size.Height == 0 {
				return 1
			}
			aspectRatio := size.Width / size.Height
			if aspectRatio > 1.2 && totalPages >= 2 {
				return 2
			}
			return 1
		}

		pdfContainer := container.NewMax()

		renderPages := func(pagesToShow int) {
			if pdfMgr == nil || totalPages == 0 {
				pdfContainer.Objects = []fyne.CanvasObject{widget.NewLabel("Failed to load PDF")}
				pdfContainer.Refresh()
				return
			}

			if pagesToShow == 2 && currentPage+1 < totalPages {
				img1, err1 := pdfMgr.GetPageImage(currentPage)
				img2, err2 := pdfMgr.GetPageImage(currentPage + 1)
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

			img, err := pdfMgr.GetPageImage(currentPage)
			if err != nil {
				pdfContainer.Objects = []fyne.CanvasObject{widget.NewLabel("Error rendering page")}
				pdfContainer.Refresh()
				return
			}
			canvasImg := canvas.NewImageFromImage(img)
			canvasImg.FillMode = canvas.ImageFillContain

			pdfContainer.Objects = []fyne.CanvasObject{canvasImg}
			pdfContainer.Refresh()
			pageLabel.SetText(fmt.Sprintf("Page %d / %d", currentPage+1, totalPages))
		}

		prevBtn := widget.NewButtonWithIcon("PREV\nPAGE", theme.NavigateBackIcon(), func() {
			if currentPage > 0 {
				currentPage -= currentPagesToShow
				if currentPage < 0 {
					currentPage = 0
				}
				renderPages(currentPagesToShow)
			}
		})
		prevBtn.Importance = widget.HighImportance

		nextBtn := widget.NewButtonWithIcon("NEXT\nPAGE", theme.NavigateNextIcon(), func() {
			if currentPage+currentPagesToShow < totalPages {
				currentPage += currentPagesToShow
				renderPages(currentPagesToShow)
			}
		})
		nextBtn.Importance = widget.HighImportance

		exitBtn := widget.NewButtonWithIcon("Exit", theme.CancelIcon(), func() {
			if metroAudio != nil {
				metroAudio.Stop()
				metroAudio.Close()
			}
			if pdfMgr != nil {
				pdfMgr.Close()
			}
			showLibrary()
		})
		exitBtn.Importance = widget.DangerImportance

		titleLabel := widget.NewLabelWithStyle(score.DisplayTitle(), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		topBarControls := container.NewHBox(exitBtn, widget.NewLabel("  "), metroIndicatorContainer)
		topBar := container.NewBorder(nil, nil, topBarControls, nil, titleLabel)

		toolsBtn := widget.NewButtonWithIcon("Tools", theme.SettingsIcon(), func() {
			ShowToolsMenu(w, metroAudio, func(cb func(bool)) {
				dialogBeatCb = cb
			})
		})

		rightSidebar := container.NewBorder(
			container.NewVBox(pageLabel, widget.NewSeparator(), toolsBtn, widget.NewSeparator()),
			nil, nil, nil,
			container.NewGridWithRows(2, prevBtn, nextBtn),
		)

		viewer := NewResponsiveViewer(func(size fyne.Size) {
			newPagesToShow := getPagesToShow(size)
			if newPagesToShow != currentPagesToShow {
				currentPagesToShow = newPagesToShow
				renderPages(currentPagesToShow)
			}
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

		renderPages(currentPagesToShow)
	}

	updateGrid = func() {
		isLoggedIn := app.Preferences().String("jwt_token") != ""
		scores, err := db.GetScores()
		if err != nil {
			dialog.ShowError(err, w)
			scores = []localdb.Score{}
		}

		grid := container.NewGridWrap(fyne.NewSize(200, 200))
		query := strings.ToLower(searchEntry.Text)

		for _, s := range scores {
			score := s

			if len(query) >= 3 && !strings.Contains(strings.ToLower(score.DisplayTitle()), query) {
				continue
			}

			cardContent := container.NewVBox(
				widget.NewLabelWithStyle(score.DisplayTitle(), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				layout.NewSpacer(),
			)
			card := widget.NewCard("", "", cardContent)

			if editMode {
				var editControls *fyne.Container

				if score.IsOwner {
					shareBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
						ShowAccessDialog(w, app, "Share Score", score.Title, "Share", func(email *string, bandID *uint32) error {
							token := app.Preferences().String("jwt_token")
							server := app.Preferences().String("server_addr")
							conn, err := network.NewGRPCClient(server, token)
							if err != nil {
								return err
							}
							defer conn.Close()
							client := scorepb.NewScoreServiceClient(conn)
							_, err = client.ShareScore(context.Background(), &scorepb.ShareScoreRequest{
								ScoreId:      score.ID,
								TargetEmail:  email,
								TargetBandId: bandID,
							})
							return err
						})
					})

					revokeBtn := widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), func() {
						ShowAccessDialog(w, app, "Revoke Score Access", score.Title, "Revoke", func(email *string, bandID *uint32) error {
							token := app.Preferences().String("jwt_token")
							server := app.Preferences().String("server_addr")
							conn, err := network.NewGRPCClient(server, token)
							if err != nil {
								return err
							}
							defer conn.Close()
							client := scorepb.NewScoreServiceClient(conn)
							_, err = client.RevokeScoreAccess(context.Background(), &scorepb.RevokeScoreAccessRequest{
								ScoreId:      score.ID,
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

					editTitleBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
						entry := NewAutoKeyboardEntry()
						entry.SetText(score.Title)
						var d dialog.Dialog
						formContent := container.NewVBox(
							widget.NewLabel("Edit Score Title:"),
							entry,
							container.NewHBox(
								layout.NewSpacer(),
								widget.NewButton("Save", func() {
									if entry.Text != "" {
										_ = db.UpdateScore(score.ID, entry.Text)
										onScoresChanged()
										updateGrid()
										d.Hide()
									}
								}),
								widget.NewButton("Cancel", func() { d.Hide() }),
							),
						)
						d = dialog.NewCustomWithoutButtons("Edit Score", formContent, w)
						d.Show()
						w.Canvas().Focus(entry)
					})

					deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
						dialog.ShowConfirm("Delete Score", fmt.Sprintf("Are you sure you want to delete '%s'?", score.Title), func(confirmed bool) {
							if confirmed {
								_ = db.MarkScoreDeleted(score.ID)
								onScoresChanged()
								updateGrid()
							}
						}, w)
					})
					deleteBtn.Importance = widget.DangerImportance

					editControls = container.NewHBox(shareBtn, revokeBtn, editTitleBtn, deleteBtn)
				} else {
					readOnlyLabel := widget.NewLabelWithStyle("Shared with you", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
					setAliasBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
						entry := NewAutoKeyboardEntry()
						entry.SetText(score.DisplayTitle())
						var d dialog.Dialog
						formContent := container.NewVBox(
							widget.NewLabel("Set local alias (only visible to you):"),
							entry,
							container.NewHBox(
								layout.NewSpacer(),
								widget.NewButton("Save", func() {
									_ = db.SetScoreAlias(score.ID, entry.Text)
									onScoresChanged()
									updateGrid()
									d.Hide()
								}),
								widget.NewButton("Clear", func() {
									_ = db.SetScoreAlias(score.ID, "")
									onScoresChanged()
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
					editControls = container.NewHBox(readOnlyLabel, setAliasBtn)
				}

				item := container.NewBorder(nil, editControls, nil, nil, card)
				grid.Add(item)
			} else {
				openBtn := widget.NewButtonWithIcon("OPEN", theme.MediaPlayIcon(), func() {
					showScore(score)
				})
				openBtn.Importance = widget.HighImportance

				item := container.NewBorder(nil, openBtn, nil, nil, card)
				grid.Add(item)
			}
		}

		gridWrapper.Objects = []fyne.CanvasObject{container.NewPadded(container.NewVScroll(grid))}
		gridWrapper.Refresh()
	}

	searchEntry.OnChanged = func(s string) {
		if len(s) >= 3 || len(s) == 0 {
			updateGrid()
		}
	}

	showLibrary = func() {
		updateGrid()

		addBtn := widget.NewButtonWithIcon("Add Score", theme.ContentAddIcon(), func() {
			fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil || reader == nil {
					return
				}
				defer reader.Close()

				filePath := reader.URI().Path()
				title := reader.URI().Name()
				_, err = db.AddScore(title, filePath)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				onScoresChanged()
				updateGrid()
			}, w)
			fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
			fileDialog.Show()
		})
		addBtn.Importance = widget.HighImportance

		editToggleBtn := widget.NewButtonWithIcon("Manage", theme.SettingsIcon(), func() {
			editMode = !editMode
			showLibrary()
		})

		if editMode {
			editToggleBtn.SetText("Done")
			editToggleBtn.SetIcon(theme.ConfirmIcon())
			editToggleBtn.Importance = widget.HighImportance
		}

		topControls := container.NewHBox(addBtn, editToggleBtn)

		backToDashBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backToDashBtn.Importance = widget.WarningImportance

		searchContainer := container.NewPadded(searchEntry)
		header := container.NewBorder(nil, nil, backToDashBtn, topControls, searchContainer)

		view := container.NewBorder(header, nil, nil, nil, gridWrapper)

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	showLibrary()
	return contentWrapper
}
