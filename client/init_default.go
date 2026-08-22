//go:build !windows && !linux

package main

import (
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/sysmock"
)

func InitManagers() (system.NetworkManager, system.PowerManager, system.MediaManager, system.DeviceManager) {
	return &sysmock.MockNetworkManager{Status: system.StatusDisconnected},
		&sysmock.MockPowerManager{BatteryLevel: 85, Charging: true},
		&sysmock.MockMediaManager{Volume: 50, Brightness: 80},
		&sysmock.MockDeviceManager{IsAwake: true}
}
