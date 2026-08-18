package sysmock

import "github.com/ziomciopoziomcio/digital-music-stand/client/system"

type MockNetworkManager struct {
	Status system.NetworkStatus
}

func (m *MockNetworkManager) GetAvailableNetworks() ([]system.Network, error) {
	return []system.Network{
		{SSID: "Philharmonic_Guest", Strength: 80, Secure: true},
		{SSID: "Concert_Hall_Stage", Strength: 100, Secure: true},
		{SSID: "Free_WiFi", Strength: 30, Secure: false},
	}, nil
}

func (m *MockNetworkManager) ConnectWiFi(ssid, password string) error {
	m.Status = system.StatusConnected
	return nil
}

func (m *MockNetworkManager) Disconnect() error {
	m.Status = system.StatusDisconnected
	return nil
}

func (m *MockNetworkManager) GetNetworkStatus() system.NetworkStatus {
	return m.Status
}

type MockPowerManager struct {
	BatteryLevel int
	Charging     bool
}

func (m *MockPowerManager) GetBatteryPercentage() (int, error) {
	return m.BatteryLevel, nil
}

func (m *MockPowerManager) IsCharging() (bool, error) {
	return m.Charging, nil
}

type MockMediaManager struct {
	Volume     int
	Brightness int
}

func (m *MockMediaManager) GetVolume() (int, error) {
	return m.Volume, nil
}

func (m *MockMediaManager) SetVolume(level int) error {
	m.Volume = level
	return nil
}

func (m *MockMediaManager) GetBrightness() (int, error) {
	return m.Brightness, nil
}

func (m *MockMediaManager) SetBrightness(level int) error {
	m.Brightness = level
	return nil
}

type MockDeviceManager struct {
	IsAwake bool
}

func (m *MockDeviceManager) Reboot() error {
	return nil
}

func (m *MockDeviceManager) Shutdown() error {
	return nil
}

func (m *MockDeviceManager) SetKeepAwake(awake bool) error {
	m.IsAwake = awake
	return nil
}

func (m *MockDeviceManager) IsKeepAwake() bool {
	return m.IsAwake
}

type MockStorageManager struct{}

func (m *MockStorageManager) GetMountedUSBDrives() ([]string, error) {
	return []string{"/media/usb/MUSIC_DRIVE_1"}, nil
}
