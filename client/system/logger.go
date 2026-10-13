package system

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func InitLogger(appDataDir string) (*os.File, error) {
	logsDir := filepath.Join(appDataDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, err
	}

	cleanupOldLogs(logsDir, 7*24*time.Hour)

	logFileName := fmt.Sprintf("dms_%s.log", time.Now().Format("2006-01-02_15-04-05"))
	logFilePath := filepath.Join(logsDir, logFileName)

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)

	os.Stdout = file
	os.Stderr = file

	return file, nil
}

func cleanupOldLogs(logsDir string, maxAge time.Duration) {
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err == nil {
			if now.Sub(info.ModTime()) > maxAge {
				os.Remove(filepath.Join(logsDir, entry.Name()))
			}
		}
	}
}
