package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
)

func BuildConcertMode(w fyne.Window, db *localdb.DBManager, goBack func(), openSetup func(editingConcert *localdb.Concert), onDeleteConcert func()) *fyne.Container {
	contentWrapper := container.NewMax()

	var showConcertList func()
	var playConcert func(concert localdb.Concert)

	showConcertList = func() {
		backBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backBtn.Importance = widget.WarningImportance

		newConcertBtn := widget.NewButtonWithIcon("New Concert", theme.ContentAddIcon(), func() {
			openSetup(nil)
		})
		newConcertBtn.Importance = widget.HighImportance

		topControls := container.NewHBox(newConcertBtn)
		header := container.NewBorder(nil, nil, backBtn, topControls,
			widget.NewLabelWithStyle("Concert Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

		concerts, err := db.GetConcerts()
		if err != nil {
			dialog.ShowError(err, w)
			concerts = []localdb.Concert{}
		}

		grid := container.NewGridWrap(fyne.NewSize(260, 220))

		for _, c := range concerts {
			concert := c

			nameLabel := widget.NewLabelWithStyle(concert.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			detailsLabel := widget.NewLabelWithStyle(fmt.Sprintf("%s\n%s\nItems: %d", concert.Location, concert.StartTime, len(concert.Items)), fyne.TextAlignCenter, fyne.TextStyle{})

			openBtn := widget.NewButtonWithIcon("ENTER", theme.MediaPlayIcon(), func() {
				playConcert(concert)
			})
			openBtn.Importance = widget.HighImportance

			editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
				openSetup(&concert)
			})

			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				dialog.ShowConfirm("Delete Concert", fmt.Sprintf("Are you sure you want to delete '%s'?", concert.Name), func(confirmed bool) {
					if confirmed {
						_ = db.MarkConcertDeleted(concert.ID)
						onDeleteConcert()
						showConcertList()
					}
				}, w)
			})
			deleteBtn.Importance = widget.DangerImportance

			actionButtons := container.NewHBox(editBtn, deleteBtn, openBtn)
			cardContent := container.NewVBox(nameLabel, detailsLabel, layout.NewSpacer())
			card := widget.NewCard("", "", cardContent)

			item := container.NewBorder(nil, actionButtons, nil, nil, card)
			grid.Add(item)
		}

		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(container.NewVScroll(grid)))
		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	playConcert = func(concert localdb.Concert) {
		if len(concert.Items) == 0 {
			dialog.ShowInformation("Empty Setlist", "This concert has no items assigned.", w)
			return
		}

		currentSongIdx := 0
		currentPage := 0
		totalPages := 0
		pagesToShow := 1

		var currentPdfMgr *pdf.Manager
		var viewerSize fyne.Size
		var activeTimerStopChan chan struct{}

		stopCurrentTimer := func() {
			if activeTimerStopChan != nil {
				close(activeTimerStopChan)
				activeTimerStopChan = nil
			}
		}

		pdfContainer := container.NewMax()
		songTitleLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		pageLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		var renderPage func()
		var loadCurrentSong func(startAtEnd bool)

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

				remainingSec := breakDuration * 60
				isTimerRunning := false

				timerClockLabel := canvas.NewText(formatTimerText(remainingSec), theme.ForegroundColor())
				timerClockLabel.Alignment = fyne.TextAlignCenter
				timerClockLabel.TextStyle = fyne.TextStyle{Bold: true}
				timerClockLabel.TextSize = 48

				timerStatusLabel := widget.NewLabelWithStyle("PAUSED", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

				var startPauseBtn *widget.Button

				startPauseBtn = widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), func() {
					if isTimerRunning {
						isTimerRunning = false
						stopCurrentTimer()
						startPauseBtn.SetText("Start")
						startPauseBtn.SetIcon(theme.MediaPlayIcon())
						timerStatusLabel.SetText("PAUSED")
					} else {
						if remainingSec <= 0 {
							return
						}
						isTimerRunning = true
						startPauseBtn.SetText("Pause")
						startPauseBtn.SetIcon(theme.MediaPauseIcon())
						timerStatusLabel.SetText("COUNTDOWN IN PROGRESS")

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
									remainingSec--
									timerClockLabel.Text = formatTimerText(remainingSec)
									timerClockLabel.Refresh()

									if remainingSec <= 0 {
										isTimerRunning = false
										timerStatusLabel.SetText("BREAK FINISHED!")
										startPauseBtn.SetText("Start")
										startPauseBtn.SetIcon(theme.MediaPlayIcon())
										return
									}
								}
							}
						}(stopCh)
					}
				})
				startPauseBtn.Importance = widget.HighImportance

				resetBtn := widget.NewButtonWithIcon("Reset", theme.ViewRefreshIcon(), func() {
					isTimerRunning = false
					stopCurrentTimer()
					remainingSec = breakDuration * 60
					timerClockLabel.Text = formatTimerText(remainingSec)
					timerClockLabel.Refresh()
					timerStatusLabel.SetText("PAUSED")
					startPauseBtn.SetText("Start")
					startPauseBtn.SetIcon(theme.MediaPlayIcon())
				})

				addMinBtn := widget.NewButton("+1 Min", func() {
					remainingSec += 60
					timerClockLabel.Text = formatTimerText(remainingSec)
					timerClockLabel.Refresh()
				})

				subMinBtn := widget.NewButton("-1 Min", func() {
					if remainingSec > 60 {
						remainingSec -= 60
					} else {
						remainingSec = 0
					}
					timerClockLabel.Text = formatTimerText(remainingSec)
					timerClockLabel.Refresh()
				})

				timerControls := container.NewHBox(
					layout.NewSpacer(),
					startPauseBtn,
					resetBtn,
					subMinBtn,
					addMinBtn,
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
			stopCurrentTimer()
			if currentPdfMgr != nil {
				currentPdfMgr.Close()
			}
			showConcertList()
		})
		exitConcertBtn.Importance = widget.DangerImportance

		prevSongBtn := widget.NewButtonWithIcon("Prev Item", theme.MediaSkipPreviousIcon(), func() {
			if currentSongIdx > 0 {
				currentSongIdx--
				loadCurrentSong(false)
			}
		})

		nextSongBtn := widget.NewButtonWithIcon("Next Item", theme.MediaSkipNextIcon(), func() {
			if currentSongIdx < len(concert.Items)-1 {
				currentSongIdx++
				loadCurrentSong(false)
			}
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
					currentSongIdx = idx
					loadCurrentSong(false)
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
			ShowToolsMenu(w)
		})

		prevPageBtn := widget.NewButtonWithIcon("PREV\nPAGE", theme.NavigateBackIcon(), func() {
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
		})
		prevPageBtn.Importance = widget.HighImportance

		nextPageBtn := widget.NewButtonWithIcon("NEXT\nPAGE", theme.NavigateNextIcon(), func() {
			if currentPage+pagesToShow < totalPages {
				currentPage += pagesToShow
				renderPage()
			} else if currentSongIdx < len(concert.Items)-1 {
				currentSongIdx++
				loadCurrentSong(false)
			}
		})
		nextPageBtn.Importance = widget.HighImportance

		rightSidebar := container.NewBorder(
			container.NewVBox(pageLabel, widget.NewSeparator(), setlistBtn, toolsBtn, widget.NewSeparator()),
			nil, nil, nil,
			container.NewGridWithRows(2, prevPageBtn, nextPageBtn),
		)

		topBar := container.NewBorder(nil, nil, exitConcertBtn, container.NewHBox(prevSongBtn, nextSongBtn), songTitleLabel)

		viewer := newResponsiveViewer(func(size fyne.Size) {
			viewerSize = size
			renderPage()
		})
		viewer.content.Objects = []fyne.CanvasObject{pdfContainer}

		mainView := container.NewBorder(
			container.NewPadded(topBar),
			nil,
			nil,
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
