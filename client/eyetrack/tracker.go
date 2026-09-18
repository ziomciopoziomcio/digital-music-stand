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

func NewTracker() *Tracker {
	return &Tracker{
		GazeChan: make(chan GazePoint, 5),
	}
}

func ensureDependencies(pyCmd string) error {
	checkCmd := exec.Command(pyCmd, "-c", "import cv2, mediapipe, numpy")
	if err := checkCmd.Run(); err != nil {
		installCmd := exec.Command(pyCmd, "-m", "pip", "install", "--upgrade", "opencv-python", "mediapipe", "numpy", "protobuf")
		return installCmd.Run()
	}
	return nil
}

func (t *Tracker) Start(cameraID int, preview bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isRunning {
		return nil
	}

	pyCmd := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		pyCmd = "python"
	}

	if err := ensureDependencies(pyCmd); err != nil {
		return fmt.Errorf("failed to install python dependencies: %v", err)
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
