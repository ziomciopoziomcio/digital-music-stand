package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
)

func BuildPracticeMode(w fyne.Window, app fyne.App, db *localdb.DBManager, onScoresChanged func(), goBack func()) *fyne.Container {
	contentWrapper := container.NewMax()

	editMode := false
	var showLibrary func()

	showScore := func(score localdb.Score) {
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
			if pdfMgr != nil {
				pdfMgr.Close()
			}
			showLibrary()
		})
		exitBtn.Importance = widget.DangerImportance

		titleLabel := widget.NewLabelWithStyle(score.Title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		topBar := container.NewBorder(nil, nil, exitBtn, nil, titleLabel)

		toolsBtn := widget.NewButtonWithIcon("Tools", theme.SettingsIcon(), func() {
			ShowToolsMenu(w)
		})

		rightSidebar := container.NewBorder(
			container.NewVBox(pageLabel, widget.NewSeparator(), toolsBtn, widget.NewSeparator()),
			nil, nil, nil,
			container.NewGridWithRows(2, prevBtn, nextBtn),
		)

		viewer := newResponsiveViewer(func(size fyne.Size) {
			newPagesToShow := getPagesToShow(size)
			if newPagesToShow != currentPagesToShow {
				currentPagesToShow = newPagesToShow
				renderPages(currentPagesToShow)
			}
		})
		viewer.content.Objects = []fyne.CanvasObject{pdfContainer}

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

	showLibrary = func() {
		scores, err := db.GetScores()
		if err != nil {
			dialog.ShowError(err, w)
			scores = []localdb.Score{}
		}

		grid := container.NewGridWrap(fyne.NewSize(200, 200))

		for _, s := range scores {
			score := s

			cardContent := container.NewVBox(
				widget.NewLabelWithStyle(score.Title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				layout.NewSpacer(),
			)
			card := widget.NewCard("", "", cardContent)

			if editMode {
				shareBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
					emailEntry := widget.NewEntry()
					emailEntry.SetPlaceHolder("user@example.com")

					var d dialog.Dialog
					shareForm := container.NewVBox(
						widget.NewLabel(fmt.Sprintf("Share score '%s':", score.Title)),
						widget.NewLabel("Enter user email:"),
						emailEntry,
						container.NewHBox(
							layout.NewSpacer(),
							widget.NewButton("Share", func() {
								if emailEntry.Text != "" {
									go func() {
										token := app.Preferences().String("jwt_token")
										server := app.Preferences().String("server_addr")
										if token == "" || server == "" {
											dialog.ShowError(fmt.Errorf("not logged in"), w)
											return
										}
										conn, err := network.NewGRPCClient(server, token)
										if err != nil {
											dialog.ShowError(err, w)
											return
										}
										defer conn.Close()

										client := scorepb.NewScoreServiceClient(conn)
										targetEmail := emailEntry.Text
										_, err = client.ShareScore(context.Background(), &scorepb.ShareScoreRequest{
											ScoreId:     score.ID,
											TargetEmail: &targetEmail,
										})
										if err != nil {
											dialog.ShowError(err, w)
										} else {
											dialog.ShowInformation("Success", "Score sharing invitation sent!", w)
										}
									}()
									d.Hide()
								}
							}),
							widget.NewButton("Cancel", func() { d.Hide() }),
						),
					)

					d = dialog.NewCustomWithoutButtons("Share Score", shareForm, w)
					d.Show()
				})

				editTitleBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
					entry := widget.NewEntry()
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
									showLibrary()
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
							showLibrary()
						}
					}, w)
				})
				deleteBtn.Importance = widget.DangerImportance

				editControls := container.NewHBox(shareBtn, editTitleBtn, deleteBtn)
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
				showLibrary()
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

		header := container.NewBorder(nil, nil, backToDashBtn, topControls, widget.NewLabelWithStyle("Practice Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(container.NewVScroll(grid)))

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	showLibrary()
	return contentWrapper
}

type responsiveViewer struct {
	widget.BaseWidget
	content  *fyne.Container
	onResize func(size fyne.Size)
	lastSize fyne.Size
}

func newResponsiveViewer(onResize func(size fyne.Size)) *responsiveViewer {
	v := &responsiveViewer{
		onResize: onResize,
		content:  container.NewMax(),
	}
	v.ExtendBaseWidget(v)
	return v
}

func (v *responsiveViewer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(v.content)
}

func (v *responsiveViewer) Resize(size fyne.Size) {
	v.BaseWidget.Resize(size)
	v.content.Resize(size)

	if v.onResize != nil && (size.Width != v.lastSize.Width || size.Height != v.lastSize.Height) {
		v.lastSize = size
		v.onResize(size)
	}
}
