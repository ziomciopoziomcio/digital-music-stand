//go:build windows

package main

import (
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/syswin"
)

func InitManagers() (system.NetworkManager, system.PowerManager, system.MediaManager, system.DeviceManager) {
	devMgr := syswin.NewWindowsDeviceManager()
	devMgr.SetKeepAwake(true)
	return syswin.NewWindowsNetworkManager(), syswin.NewWindowsPowerManager(), syswin.NewWindowsMediaManager(), devMgr
}
