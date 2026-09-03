//go:build linux

package syslinux

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
)

type LinuxNetworkManager struct {
	status system.NetworkStatus
}

func NewLinuxNetworkManager() *LinuxNetworkManager {
	return &LinuxNetworkManager{status: system.StatusDisconnected}
}

func (m *LinuxNetworkManager) GetAvailableNetworks() ([]system.Network, error) {
	log.Println("Scanning for WiFi networks (Linux)...")

	cmd := exec.Command("nmcli", "-t", "-f", "SSID,SIGNAL,SECURITY", "dev", "wifi", "list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list wifi: %v", err)
	}

	var networks []system.Network
	scanner := bufio.NewScanner(bytes.NewReader(out))
	seen := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			ssid := parts[0]
			if ssid == "" || seen[ssid] {
				continue
			}
			seen[ssid] = true

			strength, _ := strconv.Atoi(parts[1])
			secure := parts[2] != "" && parts[2] != "--"

			networks = append(networks, system.Network{
				SSID:     ssid,
				Strength: strength,
				Secure:   secure,
			})
		}
	}
	log.Printf("Found %d networks.", len(networks))
	return networks, nil
}

func (m *LinuxNetworkManager) ConnectWiFi(ssid, password string) error {
	log.Printf("Attempting to connect to WiFi: %s...", ssid)

	errChan := make(chan error, 1)

	go func() {
		cmd := exec.Command("nmcli", "dev", "wifi", "connect", ssid, "password", password)
		if err := cmd.Run(); err != nil {
			errChan <- fmt.Errorf("failed to connect to %s: %v", ssid, err)
			return
		}
		m.status = system.StatusConnected
		log.Printf("Successfully connected to WiFi: %s", ssid)
		errChan <- nil
	}()

	return <-errChan
}

func (m *LinuxNetworkManager) Disconnect() error {
	log.Println("Disconnecting from WiFi...")
	go func() {
		cmd := exec.Command("nmcli", "dev", "disconnect", "wlan0")
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to disconnect: %v", err)
		} else {
			m.status = system.StatusDisconnected
			log.Println("Disconnected from WiFi.")
		}
	}()
	return nil
}

func (m *LinuxNetworkManager) GetNetworkStatus() system.NetworkStatus {
	cmd := exec.Command("nmcli", "-t", "-f", "STATE", "general")
	out, err := cmd.Output()
	if err == nil && strings.Contains(string(out), "connected") && !strings.Contains(string(out), "disconnected") {
		return system.StatusConnected
	}
	return system.StatusDisconnected
}

type LinuxPowerManager struct{}

func NewLinuxPowerManager() *LinuxPowerManager {
	return &LinuxPowerManager{}
}

func (m *LinuxPowerManager) getBatteryPath() string {
	files, _ := filepath.Glob("/sys/class/power_supply/BAT*")
	if len(files) > 0 {
		return files[0]
	}
	return ""
}

func (m *LinuxPowerManager) GetBatteryPercentage() (int, error) {
	path := m.getBatteryPath()
	if path == "" {
		return 100, nil
	}
	data, err := os.ReadFile(filepath.Join(path, "capacity"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func (m *LinuxPowerManager) IsCharging() (bool, error) {
	path := m.getBatteryPath()
	if path == "" {
		return true, nil
	}
	data, err := os.ReadFile(filepath.Join(path, "status"))
	if err != nil {
		return false, err
	}
	status := strings.TrimSpace(string(data))
	return status == "Charging" || status == "Full", nil
}

type LinuxMediaManager struct{}

func NewLinuxMediaManager() *LinuxMediaManager {
	return &LinuxMediaManager{}
}

func (m *LinuxMediaManager) GetVolume() (int, error) {
	cmd := exec.Command("amixer", "sget", "Master")
	out, err := cmd.Output()
	if err != nil {
		return 50, nil
	}

	outStr := string(out)
	start := strings.Index(outStr, "[")
	end := strings.Index(outStr, "%]")
	if start != -1 && end != -1 && end > start {
		vol, _ := strconv.Atoi(outStr[start+1 : end])
		return vol, nil
	}
	return 50, nil
}

func (m *LinuxMediaManager) SetVolume(level int) error {
	go func() {
		cmd := exec.Command("amixer", "sset", "Master", fmt.Sprintf("%d%%", level))
		cmd.Run()
	}()
	return nil
}

func (m *LinuxMediaManager) GetBrightness() (int, error) {
	cmd := exec.Command("brightnessctl", "get")
	out, err := cmd.Output()
	if err != nil {
		return 100, nil
	}

	cmdMax := exec.Command("brightnessctl", "max")
	outMax, _ := cmdMax.Output()

	current, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	maxVal, _ := strconv.ParseFloat(strings.TrimSpace(string(outMax)), 64)

	if maxVal > 0 {
		return int((current / maxVal) * 100), nil
	}
	return 100, nil
}

func (m *LinuxMediaManager) SetBrightness(level int) error {
	go func() {
		log.Printf("Setting brightness to %d%% (Linux)...", level)
		cmd := exec.Command("brightnessctl", "set", fmt.Sprintf("%d%%", level))
		cmd.Run()
	}()
	return nil
}

type LinuxDeviceManager struct {
	keepAwake bool
}

func NewLinuxDeviceManager() *LinuxDeviceManager {
	return &LinuxDeviceManager{keepAwake: true}
}

func (m *LinuxDeviceManager) Reboot() error {
	go exec.Command("sudo", "systemctl", "reboot").Run()
	return nil
}

func (m *LinuxDeviceManager) Shutdown() error {
	go exec.Command("sudo", "systemctl", "poweroff").Run()
	return nil
}

func (m *LinuxDeviceManager) SetKeepAwake(awake bool) error {
	m.keepAwake = awake
	go func() {
		if awake {
			exec.Command("xset", "s", "off").Run()
			exec.Command("xset", "-dpms").Run()
		} else {
			exec.Command("xset", "s", "on").Run()
			exec.Command("xset", "+dpms").Run()
		}
	}()
	return nil
}

func (m *LinuxDeviceManager) IsKeepAwake() bool {
	return m.keepAwake
}

type LinuxStorageManager struct{}

func NewLinuxStorageManager() *LinuxStorageManager {
	return &LinuxStorageManager{}
}

func (m *LinuxStorageManager) GetMountedUSBDrives() ([]string, error) {
	var drives []string
	files, err := os.ReadDir("/media")
	if err != nil {
		return drives, err
	}

	for _, userDir := range files {
		if userDir.IsDir() {
			userPath := filepath.Join("/media", userDir.Name())
			subDirs, _ := os.ReadDir(userPath)
			for _, drive := range subDirs {
				if drive.IsDir() {
					drives = append(drives, filepath.Join(userPath, drive.Name()))
				}
			}
		}
	}
	return drives, nil
}
