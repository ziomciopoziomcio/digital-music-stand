package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/network"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
)

func ShowAccessDialog(
	w fyne.Window,
	app fyne.App,
	dialogTitle string,
	itemTitle string,
	actionBtnText string,
	actionFunc func(targetEmail *string, targetBandID *uint32) error,
) {
	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("user@example.com")

	bandSelect := widget.NewSelect([]string{"Loading bands..."}, nil)
	var bandMap map[string]uint32

	targetType := widget.NewRadioGroup([]string{"User (Email)", "Band"}, func(selected string) {
		if selected == "User (Email)" {
			emailEntry.Enable()
			bandSelect.Disable()
		} else {
			emailEntry.Disable()
			bandSelect.Enable()
		}
	})
	targetType.SetSelected("User (Email)")

	go func() {
		token := app.Preferences().String("jwt_token")
		server := app.Preferences().String("server_addr")
		if token != "" && server != "" {
			if conn, err := network.NewGRPCClient(server, token); err == nil {
				defer conn.Close()
				client := bandpb.NewBandServiceClient(conn)
				resp, err := client.ListMyBands(context.Background(), &bandpb.ListMyBandsRequest{})
				if err == nil {
					bandMap = make(map[string]uint32)
					var names []string
					for _, b := range resp.GetBands() {
						bandMap[b.Name] = b.Id
						names = append(names, b.Name)
					}
					if len(names) > 0 {
						bandSelect.Options = names
						bandSelect.SetSelected(names[0])
					} else {
						bandSelect.Options = []string{"No bands available"}
						bandSelect.SetSelected("No bands available")
					}
					bandSelect.Refresh()
				}
			}
		}
	}()

	var d dialog.Dialog
	form := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("%s '%s':", dialogTitle, itemTitle)),
		targetType,
		widget.NewLabel("Email address:"), emailEntry,
		widget.NewLabel("Select Band:"), bandSelect,
		container.NewHBox(
			layout.NewSpacer(),
			widget.NewButton(actionBtnText, func() {
				isUser := targetType.Selected == "User (Email)"
				if isUser && emailEntry.Text == "" {
					return
				}
				if !isUser && (bandSelect.Selected == "" || bandSelect.Selected == "Loading bands..." || bandSelect.Selected == "No bands available") {
					return
				}

				go func() {
					var targetEmail *string
					var targetBandID *uint32

					if isUser {
						targetEmail = &emailEntry.Text
					} else {
						bid := bandMap[bandSelect.Selected]
						targetBandID = &bid
					}

					if err := actionFunc(targetEmail, targetBandID); err != nil {
						dialog.ShowError(err, w)
					} else {
						dialog.ShowInformation("Success", "Operation completed successfully!", w)
					}
				}()
				d.Hide()
			}),
			widget.NewButton("Cancel", func() { d.Hide() }),
		),
	)

	d = dialog.NewCustomWithoutButtons(dialogTitle, form, w)
	d.Show()
}
