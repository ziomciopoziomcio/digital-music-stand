package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
)

func BuildInbox(w fyne.Window, db *localdb.DBManager, goBack func(), onAction func(notif localdb.Notification, accept bool)) *fyne.Container {
	contentWrapper := container.NewMax()

	var renderList func()

	renderList = func() {
		notifications, err := db.GetPendingNotifications()
		if err != nil {
			notifications = []localdb.Notification{}
		}

		if len(notifications) == 0 {
			emptyLabel := widget.NewLabelWithStyle("Your inbox is empty.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

			backBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
			backBtn.Importance = widget.WarningImportance
			header := container.NewBorder(nil, nil, backBtn, nil, widget.NewLabelWithStyle("Inbox", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

			view := container.NewBorder(header, nil, nil, nil, container.NewCenter(emptyLabel))
			contentWrapper.Objects = []fyne.CanvasObject{view}
			contentWrapper.Refresh()
			return
		}

		list := widget.NewList(
			func() int { return len(notifications) },
			func() fyne.CanvasObject {
				title := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
				body := widget.NewLabel("")

				acceptBtn := widget.NewButtonWithIcon("", theme.ConfirmIcon(), nil)
				acceptBtn.Importance = widget.HighImportance

				declineBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), nil)
				declineBtn.Importance = widget.DangerImportance

				actions := container.NewHBox(acceptBtn, declineBtn)
				texts := container.NewVBox(title, body)

				return container.NewBorder(nil, nil, nil, actions, texts)
			},
			func(i widget.ListItemID, o fyne.CanvasObject) {
				notif := notifications[i]
				c := o.(*fyne.Container)

				var title *widget.Label
				var body *widget.Label
				var actions *fyne.Container

				for _, obj := range c.Objects {
					if vBox, ok := obj.(*fyne.Container); ok && len(vBox.Objects) == 2 {
						if l, ok := vBox.Objects[0].(*widget.Label); ok {
							title = l
						}
						if l, ok := vBox.Objects[1].(*widget.Label); ok {
							body = l
						}
					}
					if hBox, ok := obj.(*fyne.Container); ok && len(hBox.Objects) == 2 {
						actions = hBox
					}
				}

				if title != nil {
					title.SetText(notif.Title)
				}
				if body != nil {
					body.SetText(notif.Body)
				}

				if actions != nil {
					acceptBtn := actions.Objects[0].(*widget.Button)
					declineBtn := actions.Objects[1].(*widget.Button)

					acceptBtn.OnTapped = func() {
						onAction(notif, true)
						_ = db.ResolveNotification(notif.ID, "accepted")
						renderList()
					}

					declineBtn.OnTapped = func() {
						onAction(notif, false)
						_ = db.ResolveNotification(notif.ID, "declined")
						renderList()
					}
				}
			},
		)

		backBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backBtn.Importance = widget.WarningImportance

		refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), renderList)

		header := container.NewBorder(nil, nil, backBtn, refreshBtn, widget.NewLabelWithStyle("Inbox", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(list))

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	renderList()
	return contentWrapper
}
