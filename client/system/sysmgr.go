package system

type NetworkStatus string

const (
	StatusConnected    NetworkStatus = "connected"
	StatusDisconnected NetworkStatus = "disconnected"
)

type Network struct {
	SSID     string
	Strength int
	Secure   bool
}

type NetworkManager interface {
	GetAvailableNetworks() ([]Network, error)
	ConnectWiFi(ssid, password string) error
	Disconnect() error
	GetNetworkStatus() NetworkStatus
}

type PowerManager interface {
	GetBatteryPercentage() (int, error)
	IsCharging() (bool, error)
}

type MediaManager interface {
	GetVolume() (int, error)
	SetVolume(level int) error
	GetBrightness() (int, error)
	SetBrightness(level int) error
}

type DeviceManager interface {
	Reboot() error
	Shutdown() error
	SetKeepAwake(awake bool) error
	IsKeepAwake() bool
}

type StorageManager interface {
	GetMountedUSBDrives() ([]string, error)
}
