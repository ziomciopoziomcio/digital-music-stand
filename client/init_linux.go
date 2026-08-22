//go:build linux

package main

import (
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/syslinux"
)

func InitManagers() (system.NetworkManager, system.PowerManager, system.MediaManager, system.DeviceManager) {
	devMgr := syslinux.NewLinuxDeviceManager()
	devMgr.SetKeepAwake(true)
	return syslinux.NewLinuxNetworkManager(), syslinux.NewLinuxPowerManager(), syslinux.NewLinuxMediaManager(), devMgr
}
