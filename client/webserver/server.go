package webserver

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var staticFiles embed.FS

type State string

const (
	StateIdle      State = "idle"
	StatePending   State = "pending"
	StateConnected State = "connected"
)

type Manager struct {
	mu         sync.Mutex
	currentPIN string
	Status     State
	UploadDir  string
}

func NewManager(uploadDir string) *Manager {
	os.MkdirAll(uploadDir, 0755)
	return &Manager{
		UploadDir: uploadDir,
		Status:    StateIdle,
	}
}

func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func (m *Manager) Start(port int) error {
	mux := http.NewServeMux()
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	mux.HandleFunc("/api/request_pin", m.handleRequestPin)
	mux.HandleFunc("/api/status", m.handleStatus)
	mux.HandleFunc("/api/upload", m.handleUpload)

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	go http.ListenAndServe(addr, mux)
	return nil
}

func (m *Manager) ConfirmPIN(pin string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Status == StatePending && m.currentPIN == pin {
		m.Status = StateConnected
		return true
	}
	return false
}

func (m *Manager) handleRequestPin(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n, _ := rand.Int(rand.Reader, big.NewInt(10000))
	m.currentPIN = fmt.Sprintf("%04d", n.Int64())
	m.Status = StatePending

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"pin": m.currentPIN})
}

func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": string(m.Status)})
}

func (m *Manager) handleUpload(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	status := m.Status
	m.mu.Unlock()

	if status != StateConnected {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("pdf_file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dstPath := filepath.Join(m.UploadDir, handler.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	io.Copy(dst, file)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully"})
}
