//go:build !windows && !linux

package main

import (
	"fyne.io/fyne/v2"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system/sysmock"
)

func InitManagers() (system.NetworkManager, system.PowerManager, system.MediaManager, system.DeviceManager) {
	return &sysmock.MockNetworkManager{Status: system.StatusDisconnected},
		&sysmock.MockPowerManager{BatteryLevel: 85, Charging: true},
		&sysmock.MockMediaManager{Volume: 50, Brightness: 80},
		&sysmock.MockDeviceManager{IsAwake: true}
}

// InitPlatformWindow is a no-op on non-Windows/non-Linux platforms.
func InitPlatformWindow(_ fyne.Window) {}
