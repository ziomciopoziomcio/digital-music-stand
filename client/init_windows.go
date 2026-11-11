//go:build windows

package main

import (
	"fyne.io/fyne/v2"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/syswin"
)

func InitManagers() (system.NetworkManager, system.PowerManager, system.MediaManager, system.DeviceManager) {
	devMgr := syswin.NewWindowsDeviceManager()
	devMgr.SetKeepAwake(true)
	return syswin.NewWindowsNetworkManager(), syswin.NewWindowsPowerManager(), syswin.NewWindowsMediaManager(), devMgr
}

// InitPlatformWindow initializes Windows-specific window features including touchscreen interaction hook.
func InitPlatformWindow(w fyne.Window) {
	syswin.HookWindowsTouch(w)
}
