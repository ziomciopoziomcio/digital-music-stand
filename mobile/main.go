package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.NewWithID("com.digitalmusicstand.mobile")
	myWindow := myApp.NewWindow("Remote Control")
	myWindow.Resize(fyne.NewSize(360, 800))

	var showConnect func()
	var showMain func(ip, pin string)
	var showScoreViewer func(ip, pin, scoreID, title string)

	httpClient := &http.Client{Timeout: 3 * time.Second}

	discoverTablet := func() (string, string) {
		conn, err := net.ListenPacket("udp4", ":0")
		if err != nil {
			return "", ""
		}
		defer conn.Close()

		addr, _ := net.ResolveUDPAddr("udp4", "255.255.255.255:8090")
		conn.WriteTo([]byte("DMS_DISCOVER"), addr)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))

		buf := make([]byte, 1024)
		n, srvAddr, err := conn.ReadFrom(buf)
		if err == nil {
			ip := strings.Split(srvAddr.String(), ":")[0]
			return ip, string(buf[:n])
		}
		return "", ""
	}

	showConnect = func() {
		ipEntry := widget.NewEntry()
		ipEntry.SetPlaceHolder("192.168.1.50")
		ipEntry.SetText(myApp.Preferences().String("local_ip"))

		pinEntry := widget.NewEntry()
		pinEntry.SetPlaceHolder("PIN")
		pinEntry.SetText(myApp.Preferences().String("local_pin"))

		connectManualBtn := widget.NewButton("Connect Manually", func() {
			myApp.Preferences().SetString("local_ip", ipEntry.Text)
			myApp.Preferences().SetString("local_pin", pinEntry.Text)
			showMain(ipEntry.Text, pinEntry.Text)
		})

		autoBtn := widget.NewButtonWithIcon("Auto Discover Tablet", theme.SearchIcon(), func() {
			prog := dialog.NewCustomWithoutButtons("Searching...", container.NewPadded(widget.NewProgressBarInfinite()), myWindow)
			prog.Show()
			go func() {
				ip, pin := discoverTablet()
				prog.Hide()
				if ip != "" && pin != "" {
					myApp.Preferences().SetString("local_ip", ip)
					myApp.Preferences().SetString("local_pin", pin)
					showMain(ip, pin)
				} else {
					dialog.ShowInformation("Not Found", "Tablet not found on local network.", myWindow)
				}
			}()
		})
		autoBtn.Importance = widget.HighImportance

		form := container.NewVBox(
			widget.NewLabelWithStyle("Remote Control", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabel(""),
			autoBtn,
			widget.NewLabel(""),
			widget.NewSeparator(),
			widget.NewLabel("Manual Connection:"),
			ipEntry,
			pinEntry,
			connectManualBtn,
		)
		myWindow.SetContent(container.NewPadded(form))
	}

	showScoreViewer = func(ip, pin, scoreID, title string) {
		page := 0
		imgCanvas := canvas.NewImageFromResource(nil)
		imgCanvas.FillMode = canvas.ImageFillContain

		pageLabel := widget.NewLabelWithStyle(fmt.Sprintf("Page %d", page+1), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		loadPage := func() {
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
		}

		prevBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
			if page > 0 {
				page--
				loadPage()
			}
		})
		nextBtn := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
			page++
			loadPage()
		})
		closeBtn := widget.NewButtonWithIcon("Back", theme.CancelIcon(), func() { showMain(ip, pin) })

		topBar := container.NewBorder(nil, nil, closeBtn, nil, widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		bottomBar := container.NewBorder(nil, nil, prevBtn, nextBtn, pageLabel)

		loadPage()
		myWindow.SetContent(container.NewBorder(topBar, bottomBar, nil, nil, imgCanvas))
	}

	showMain = func(ip, pin string) {
		fastClient := &http.Client{Timeout: 500 * time.Millisecond}

		sendCommand := func(action string) {
			url := fmt.Sprintf("http://%s:8089/api/command", ip)
			payload, _ := json.Marshal(map[string]interface{}{"action": action, "value": 0})
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
			req.Header.Set("X-Remote-PIN", pin)
			fastClient.Do(req)
		}

		infoLabel := widget.NewLabelWithStyle("Fetching state...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		pageInfoLabel := widget.NewLabelWithStyle("-", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
		stopPolling := make(chan struct{})

		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopPolling:
					return
				case <-ticker.C:
					url := fmt.Sprintf("http://%s:8089/api/state", ip)
					req, _ := http.NewRequest("GET", url, nil)
					req.Header.Set("X-Remote-PIN", pin)

					if resp, err := fastClient.Do(req); err == nil && resp.StatusCode == 200 {
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
				widget.NewButtonWithIcon("Prev Item", theme.MediaSkipPreviousIcon(), func() { sendCommand("PREV_ITEM") }),
				widget.NewButtonWithIcon("Next Item", theme.MediaSkipNextIcon(), func() { sendCommand("NEXT_ITEM") }),
				widget.NewButtonWithIcon("Prev Page", theme.NavigateBackIcon(), func() { sendCommand("PREV_PAGE") }),
				widget.NewButtonWithIcon("Next Page", theme.NavigateNextIcon(), func() { sendCommand("NEXT_PAGE") }),
			),
			widget.NewLabel(""),
			container.NewGridWithColumns(2,
				widget.NewButtonWithIcon("Toggle Timer", theme.HistoryIcon(), func() { sendCommand("TOGGLE_TIMER") }),
				widget.NewButtonWithIcon("Lock Tablet", theme.LogoutIcon(), func() { sendCommand("LOCK_SCREEN") }),
			),
		)

		libraryList := widget.NewList(
			func() int { return 0 },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(i widget.ListItemID, o fyne.CanvasObject) {},
		)

		libraryContent := container.NewBorder(
			widget.NewButtonWithIcon("Refresh Scores", theme.ViewRefreshIcon(), func() {
				url := fmt.Sprintf("http://%s:8089/api/scores", ip)
				req, _ := http.NewRequest("GET", url, nil)
				req.Header.Set("X-Remote-PIN", pin)

				if resp, err := httpClient.Do(req); err == nil && resp.StatusCode == 200 {
					var scores []map[string]interface{}
					json.NewDecoder(resp.Body).Decode(&scores)

					libraryList.Length = func() int { return len(scores) }
					libraryList.CreateItem = func() fyne.CanvasObject {
						return widget.NewButton("", nil)
					}
					libraryList.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
						btn := o.(*widget.Button)
						scoreID := scores[i]["ID"].(string)
						title := scores[i]["Title"].(string)
						btn.SetText(title)
						btn.OnTapped = func() {
							close(stopPolling)
							showScoreViewer(ip, pin, scoreID, title)
						}
					}
					libraryList.Refresh()
				}
			}),
			nil, nil, nil,
			libraryList,
		)

		tabs := container.NewAppTabs(
			container.NewTabItemWithIcon("Remote", theme.ComputerIcon(), container.NewPadded(remoteContent)),
			container.NewTabItemWithIcon("Library", theme.DocumentIcon(), container.NewPadded(libraryContent)),
		)

		disconnectBtn := widget.NewButtonWithIcon("Disconnect", theme.CancelIcon(), func() {
			close(stopPolling)
			showConnect()
		})
		disconnectBtn.Importance = widget.DangerImportance

		mainLayout := container.NewBorder(nil, disconnectBtn, nil, nil, tabs)
		myWindow.SetContent(mainLayout)
	}

	showConnect()
	myWindow.ShowAndRun()
}
