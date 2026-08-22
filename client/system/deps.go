package system

import (
	"io"
	"os/exec"
	"runtime"
)

func CheckMissingDependencies() []string {
	if runtime.GOOS != "linux" {
		return nil
	}

	requiredCommands := []string{
		"nmcli",
		"amixer",
		"brightnessctl",
		"xset",
		"matchbox-keyboard",
	}

	var missing []string
	for _, cmd := range requiredCommands {
		if _, err := exec.LookPath(cmd); err != nil {
			missing = append(missing, cmd)
		}
	}

	return missing
}

func InstallDependencies(password string, missingCmds []string) error {
	if runtime.GOOS != "linux" || len(missingCmds) == 0 {
		return nil
	}

	var packagesToInstall []string
	for _, dep := range missingCmds {
		switch dep {
		case "nmcli":
			packagesToInstall = append(packagesToInstall, "network-manager")
		case "amixer":
			packagesToInstall = append(packagesToInstall, "alsa-utils")
		case "xset":
			packagesToInstall = append(packagesToInstall, "x11-xserver-utils")
		default:
			packagesToInstall = append(packagesToInstall, dep)
		}
	}

	cmd1 := exec.Command("sudo", "-S", "apt-get", "update")
	stdin1, _ := cmd1.StdinPipe()
	go func() {
		defer stdin1.Close()
		io.WriteString(stdin1, password+"\n")
	}()
	_ = cmd1.Run()

	args := append([]string{"-S", "apt-get", "install", "-y"}, packagesToInstall...)
	cmd2 := exec.Command("sudo", args...)

	stdin2, _ := cmd2.StdinPipe()
	go func() {
		defer stdin2.Close()
		io.WriteString(stdin2, password+"\n")
	}()

	return cmd2.Run()
}
