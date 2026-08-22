package syswin

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
)

type WindowsNetworkManager struct {
	status system.NetworkStatus
}

func NewWindowsNetworkManager() *WindowsNetworkManager {
	return &WindowsNetworkManager{status: system.StatusDisconnected}
}

func (m *WindowsNetworkManager) GetAvailableNetworks() ([]system.Network, error) {
	cmd := exec.Command("netsh", "wlan", "show", "networks", "mode=bssid")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var networks []system.Network
	lines := strings.Split(string(out), "\n")

	var currentSSID string
	var currentSecure bool
	var currentStrength int

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "SSID name") {
			if currentSSID != "" {
				networks = append(networks, system.Network{
					SSID:     currentSSID,
					Strength: currentStrength,
					Secure:   currentSecure,
				})
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentSSID = strings.TrimSpace(parts[1])
			}
			currentStrength = 0
			currentSecure = true

		} else if strings.HasPrefix(line, "Authentication") || strings.HasPrefix(line, "Uwierzytelnianie") {
			if strings.Contains(line, "Open") || strings.Contains(line, "Otwarte") {
				currentSecure = false
			}

		} else if strings.HasPrefix(line, "Signal") || strings.HasPrefix(line, "Sygnał") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				sigStr := strings.TrimSpace(strings.ReplaceAll(parts[1], "%", ""))
				if sig, err := strconv.Atoi(sigStr); err == nil {
					if sig > currentStrength {
						currentStrength = sig
					}
				}
			}
		}
	}

	if currentSSID != "" {
		networks = append(networks, system.Network{
			SSID:     currentSSID,
			Strength: currentStrength,
			Secure:   currentSecure,
		})
	}

	return networks, nil
}

func (m *WindowsNetworkManager) ConnectWiFi(ssid, password string) error {
	if password != "" {
		xmlProfile := fmt.Sprintf(`<?xml version="1.0"?>
<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">
    <name>%[1]s</name>
    <SSIDConfig>
        <SSID>
            <name>%[1]s</name>
        </SSID>
    </SSIDConfig>
    <connectionType>ESS</connectionType>
    <connectionMode>auto</connectionMode>
    <MSM>
        <security>
            <authEncryption>
                <authentication>WPA2PSK</authentication>
                <encryption>AES</encryption>
                <useOneX>false</useOneX>
            </authEncryption>
            <sharedKey>
                <keyType>passPhrase</keyType>
                <protected>false</protected>
                <keyMaterial>%[2]s</keyMaterial>
            </sharedKey>
        </security>
    </MSM>
</WLANProfile>`, ssid, password)

		tmpFile, err := os.CreateTemp("", "wifi-*.xml")
		if err != nil {
			return fmt.Errorf("could not create temp xml file: %v", err)
		}
		tmpName := tmpFile.Name()
		defer os.Remove(tmpName)

		if _, err := tmpFile.WriteString(xmlProfile); err != nil {
			tmpFile.Close()
			return err
		}
		tmpFile.Close()

		addCmd := exec.Command("netsh", "wlan", "add", "profile", fmt.Sprintf("filename=%s", tmpName))
		if err := addCmd.Run(); err != nil {
			return fmt.Errorf("failed to add wifi profile via netsh: %v", err)
		}
	}

	connCmd := exec.Command("netsh", "wlan", "connect", fmt.Sprintf("name=%s", ssid))
	if err := connCmd.Run(); err != nil {
		return fmt.Errorf("failed to execute connection: %v", err)
	}

	m.status = system.StatusConnected
	return nil
}

func (m *WindowsNetworkManager) Disconnect() error {
	cmd := exec.Command("netsh", "wlan", "disconnect")
	_ = cmd.Run()
	m.status = system.StatusDisconnected
	return nil
}

func (m *WindowsNetworkManager) GetNetworkStatus() system.NetworkStatus {
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	out, err := cmd.Output()
	outStr := strings.ToLower(string(out))
	if err == nil && (strings.Contains(outStr, "connected") || strings.Contains(outStr, "połącz")) {
		return system.StatusConnected
	}
	return system.StatusDisconnected
}

type WindowsPowerManager struct{}

func NewWindowsPowerManager() *WindowsPowerManager {
	return &WindowsPowerManager{}
}

func (m *WindowsPowerManager) GetBatteryPercentage() (int, error) {
	cmd := exec.Command("powershell", "-Command", "(Get-CimInstance -ClassName Win32_Battery).EstimatedChargeRemaining")
	out, err := cmd.Output()
	if err != nil {
		return 100, nil
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return 100, nil
	}
	return strconv.Atoi(val)
}

func (m *WindowsPowerManager) IsCharging() (bool, error) {
	cmd := exec.Command("powershell", "-Command", "(Get-CimInstance -ClassName Win32_Battery).BatteryStatus")
	out, err := cmd.Output()
	if err != nil {
		return true, nil
	}
	status := strings.TrimSpace(string(out))
	return status == "2", nil
}

type WindowsMediaManager struct {
	virtualVolume int
}

func NewWindowsMediaManager() *WindowsMediaManager {
	return &WindowsMediaManager{virtualVolume: 50}
}

func (m *WindowsMediaManager) GetVolume() (int, error) {
	return m.virtualVolume, nil
}

func (m *WindowsMediaManager) SetVolume(level int) error {
	m.virtualVolume = level
	return nil
}

func (m *WindowsMediaManager) GetBrightness() (int, error) {
	cmd := exec.Command("powershell", "-Command", "(Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness).CurrentBrightness")
	out, err := cmd.Output()
	if err != nil {
		return 100, nil
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return 100, nil
	}
	return strconv.Atoi(val)
}

func (m *WindowsMediaManager) SetBrightness(level int) error {
	cmd := exec.Command("powershell", "-Command", fmt.Sprintf("(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1, %d)", level))
	return cmd.Run()
}

type WindowsDeviceManager struct {
	keepAwake bool
}

func NewWindowsDeviceManager() *WindowsDeviceManager {
	return &WindowsDeviceManager{keepAwake: false}
}

func (m *WindowsDeviceManager) Reboot() error {
	return exec.Command("shutdown", "/r", "/t", "0").Run()
}

func (m *WindowsDeviceManager) Shutdown() error {
	return exec.Command("shutdown", "/s", "/t", "0").Run()
}

func (m *WindowsDeviceManager) SetKeepAwake(awake bool) error {
	m.keepAwake = awake

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setThreadExecutionState := kernel32.NewProc("SetThreadExecutionState")

	const (
		ES_CONTINUOUS       = 0x80000000
		ES_SYSTEM_REQUIRED  = 0x00000001
		ES_DISPLAY_REQUIRED = 0x00000002
	)

	if awake {
		setThreadExecutionState.Call(ES_CONTINUOUS | ES_DISPLAY_REQUIRED | ES_SYSTEM_REQUIRED)
	} else {
		setThreadExecutionState.Call(ES_CONTINUOUS)
	}
	return nil
}

func (m *WindowsDeviceManager) IsKeepAwake() bool {
	return m.keepAwake
}

type WindowsStorageManager struct{}

func NewWindowsStorageManager() *WindowsStorageManager {
	return &WindowsStorageManager{}
}

func (m *WindowsStorageManager) GetMountedUSBDrives() ([]string, error) {
	cmd := exec.Command("powershell", "-Command", "Get-CimInstance Win32_LogicalDisk | Where-Object DriveType -eq 2 | Select-Object -ExpandProperty DeviceID")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var drives []string
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		drive := strings.TrimSpace(line)
		if drive != "" {
			drives = append(drives, drive+"\\")
		}
	}
	return drives, nil
}
