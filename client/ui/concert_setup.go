package ui

import (
	"encoding/json"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
)

type Concert struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Location  string          `json:"location"`
	StartTime string          `json:"start_time"`
	Setlist   []localdb.Score `json:"setlist"`
}

func BuildConcertSetup(w fyne.Window, db *localdb.DBManager, goBack func()) *fyne.Container {
	contentWrapper := container.NewMax()

	var setlist []localdb.Score

	availableScores := []localdb.Score{
		{ID: 1, Title: "Imperial March", FilePath: "/mock/path1"},
		{ID: 2, Title: "Symphony No. 5", FilePath: "/mock/path2"},
		{ID: 3, Title: "Bohemian Rhapsody", FilePath: "/mock/path3"},
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("e.g. Winter Concert")

	locEntry := widget.NewEntry()
	locEntry.SetPlaceHolder("e.g. Philharmonic Hall")

	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("YYYY-MM-DD HH:MM")
	dateEntry.SetText(time.Now().Format("2006-01-02 15:00"))

	form := widget.NewForm(
		widget.NewFormItem("Event Name", nameEntry),
		widget.NewFormItem("Location", locEntry),
		widget.NewFormItem("Date & Time", dateEntry),
	)

	var leftList *widget.List
	var rightList *widget.List

	leftList = widget.NewList(
		func() int { return len(availableScores) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), nil)
			return container.NewBorder(nil, nil, nil, addBtn, title)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			score := availableScores[i]
			c := o.(*fyne.Container)

			var title *widget.Label
			var addBtn *widget.Button
			for _, obj := range c.Objects {
				if l, ok := obj.(*widget.Label); ok {
					title = l
				}
				if b, ok := obj.(*widget.Button); ok {
					addBtn = b
				}
			}

			if title != nil {
				title.SetText(score.Title)
			}
			if addBtn != nil {
				addBtn.OnTapped = func() {
					setlist = append(setlist, score)
					rightList.Refresh()
				}
			}
		},
	)

	rightList = widget.NewList(
		func() int { return len(setlist) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			upBtn := widget.NewButtonWithIcon("", theme.MoveUpIcon(), nil)
			downBtn := widget.NewButtonWithIcon("", theme.MoveDownIcon(), nil)
			delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)

			buttons := container.NewHBox(upBtn, downBtn, delBtn)
			return container.NewBorder(nil, nil, nil, buttons, title)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			score := setlist[i]
			c := o.(*fyne.Container)

			var title *widget.Label
			var buttons *fyne.Container
			for _, obj := range c.Objects {
				if l, ok := obj.(*widget.Label); ok {
					title = l
				}
				if b, ok := obj.(*fyne.Container); ok {
					buttons = b
				}
			}

			if title != nil {
				title.SetText(fmt.Sprintf("%d. %s", i+1, score.Title))
			}

			if buttons != nil {
				upBtn := buttons.Objects[0].(*widget.Button)
				downBtn := buttons.Objects[1].(*widget.Button)
				delBtn := buttons.Objects[2].(*widget.Button)

				upBtn.OnTapped = func() {
					if i > 0 {
						setlist[i], setlist[i-1] = setlist[i-1], setlist[i]
						rightList.Refresh()
					}
				}
				downBtn.OnTapped = func() {
					if int(i) < len(setlist)-1 {
						setlist[i], setlist[i+1] = setlist[i+1], setlist[i]
						rightList.Refresh()
					}
				}
				delBtn.OnTapped = func() {
					setlist = append(setlist[:i], setlist[i+1:]...)
					rightList.Refresh()
				}
			}
		},
	)

	listsContainer := container.NewGridWithColumns(2,
		container.NewBorder(
			widget.NewLabelWithStyle("Score Library", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			nil, nil, nil, leftList,
		),
		container.NewBorder(
			widget.NewLabelWithStyle("Setlist Order", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			nil, nil, nil, rightList,
		),
	)

	saveBtn := widget.NewButtonWithIcon("Save Concert (Offline)", theme.DocumentSaveIcon(), func() {
		if nameEntry.Text == "" || len(setlist) == 0 {
			dialog.ShowError(fmt.Errorf("please provide event name and add at least one score"), w)
			return
		}

		newConcert := Concert{
			ID:        fmt.Sprintf("conc-%d", time.Now().Unix()),
			Name:      nameEntry.Text,
			Location:  locEntry.Text,
			StartTime: dateEntry.Text,
			Setlist:   setlist,
		}

		jsonData, _ := json.MarshalIndent(newConcert, "", "  ")
		fmt.Printf("Saved concert:\n%s\n", string(jsonData))

		dialog.ShowInformation("Success", "Concert saved locally", w)
		goBack()
	})
	saveBtn.Importance = widget.HighImportance

	backToDashBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
	backToDashBtn.Importance = widget.WarningImportance

	header := container.NewBorder(nil, nil, backToDashBtn, nil, widget.NewLabelWithStyle("Concert Setup", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

	mainLayout := container.NewBorder(
		container.NewPadded(form),
		container.NewPadded(saveBtn),
		nil, nil,
		container.NewPadded(listsContainer),
	)

	view := container.NewBorder(header, nil, nil, nil, mainLayout)

	contentWrapper.Objects = []fyne.CanvasObject{view}
	contentWrapper.Refresh()

	return contentWrapper
}
