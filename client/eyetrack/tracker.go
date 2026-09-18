package eyetrack

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
)

//go:embed tracker.py
var trackerScript []byte

type GazePoint struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Frame string  `json:"frame"`
}

type Tracker struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	GazeChan  chan GazePoint
	isRunning bool
}

var instance *Tracker
var once sync.Once

func GetTracker() *Tracker {
	once.Do(func() {
		instance = &Tracker{
			GazeChan: make(chan GazePoint, 5),
		}
	})
	return instance
}

func getPythonCommand() string {
	if customPath := os.Getenv("DMS_PYTHON_PATH"); customPath != "" {
		return customPath
	}
	if _, err := exec.LookPath("python3.11"); err == nil {
		return "python3.11"
	}
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}
	if _, err := exec.LookPath("python"); err == nil {
		return "python"
	}
	return "python3.11"
}

func GetAvailableCameras() []string {
	pyCmd := getPythonCommand()
	script := `import os, cv2, json
os.environ["OPENCV_LOG_LEVEL"] = "SILENT"
cams = []
for i in range(5):
	cap = cv2.VideoCapture(i)
	if cap.isOpened():
		cams.append(f"Camera {i}")
		cap.release()
print(json.dumps(cams))`

	out, err := exec.Command(pyCmd, "-c", script).Output()
	var cams []string
	if err == nil {
		json.Unmarshal(out, &cams)
	}
	if len(cams) == 0 {
		cams = []string{"Camera 0"}
	}
	return cams
}

func ensureDependencies(pyCmd string) error {
	checkScript := "import cv2, numpy, PIL, mediapipe; import mediapipe.solutions.face_mesh; import sys; sys.exit(0 if mediapipe.__version__ == '0.10.21' else 1)"
	checkCmd := exec.Command(pyCmd, "-c", checkScript)
	if err := checkCmd.Run(); err == nil {
		return nil
	}

	installArgs := []string{"-m", "pip", "install", "opencv-python", "mediapipe==0.10.21", "protobuf", "numpy", "Pillow"}
	installCmd := exec.Command(pyCmd, installArgs...)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if errInstall := installCmd.Run(); errInstall != nil {
		installArgs = append(installArgs, "--break-system-packages")
		fallbackCmd := exec.Command(pyCmd, installArgs...)
		fallbackCmd.Stdout = os.Stdout
		fallbackCmd.Stderr = os.Stderr
		if errFallback := fallbackCmd.Run(); errFallback != nil {
			return fmt.Errorf("pip install failed: %v", errFallback)
		}
	}
	return nil
}

func (t *Tracker) Start(cameraID int, preview bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isRunning {
		return nil
	}

	pyCmd := getPythonCommand()

	if err := ensureDependencies(pyCmd); err != nil {
		return fmt.Errorf("failed to install dependencies: %v", err)
	}

	tempDir := os.TempDir()
	scriptPath := filepath.Join(tempDir, "dms_tracker.py")
	if err := os.WriteFile(scriptPath, trackerScript, 0644); err != nil {
		return err
	}

	prevArg := "0"
	if preview {
		prevArg = "1"
	}

	t.cmd = exec.Command(pyCmd, scriptPath, "--camera", strconv.Itoa(cameraID), "--preview", prevArg)
	t.cmd.Stderr = os.Stderr

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := t.cmd.Start(); err != nil {
		return err
	}
	t.isRunning = true

	go func() {
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 2*1024*1024)
		for scanner.Scan() {
			var p GazePoint
			if err := json.Unmarshal(scanner.Bytes(), &p); err == nil {
				select {
				case t.GazeChan <- p:
				default:
				}
			}
		}
		t.mu.Lock()
		t.isRunning = false
		t.mu.Unlock()
	}()
	return nil
}

func (t *Tracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
	t.isRunning = false
}
