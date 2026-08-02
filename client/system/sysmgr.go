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
