package remote

import (
	"encoding/json"
	"fmt"
	"image/png"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
)

type ConcertState struct {
	ConcertName    string   `json:"concert_name"`
	CurrentItemIdx int      `json:"current_item_idx"`
	CurrentPage    int      `json:"current_page"`
	TotalPages     int      `json:"total_pages"`
	IsTimerRunning bool     `json:"is_timer_running"`
	TimerSeconds   int      `json:"timer_seconds"`
	IsLocked       bool     `json:"is_locked"`
	Setlist        []string `json:"setlist"`
	CurrentScoreID string   `json:"current_score_id"`
	CurrentPDFPath string   `json:"-"`
}

type Command struct {
	Action  string `json:"action"`
	Value   int    `json:"value"`
	Payload string `json:"payload"`
}

type Server struct {
	mu          sync.RWMutex
	PIN         string
	State       ConcertState
	CommandChan chan Command
	server      *http.Server
	port        int
	db          *localdb.DBManager
}

func NewServer(port int, db *localdb.DBManager) *Server {
	rand.Seed(time.Now().UnixNano())
	return &Server{
		PIN:         fmt.Sprintf("%04d", rand.Intn(10000)),
		CommandChan: make(chan Command, 20),
		port:        port,
		db:          db,
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/command", s.handleCommand)
	mux.HandleFunc("/api/scores", s.handleScores)
	mux.HandleFunc("/api/render", s.handleRender)

	s.server = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", s.port), Handler: mux}

	go s.server.ListenAndServe()
	go s.startUDPDiscovery()
}

func (s *Server) startUDPDiscovery() {
	addr, err := net.ResolveUDPAddr("udp4", ":8090")
	if err != nil {
		return
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err == nil && string(buf[:n]) == "DMS_DISCOVER" {
			conn.WriteToUDP([]byte(s.GetPIN()), remoteAddr)
		}
	}
}

func (s *Server) SetState(state ConcertState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
}

func (s *Server) GetPIN() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PIN
}

func (s *Server) auth(w http.ResponseWriter, r *http.Request) bool {
	pin := r.Header.Get("X-Remote-PIN")
	s.mu.RLock()
	valid := (pin == s.PIN)
	s.mu.RUnlock()
	if !valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
	return valid
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.State)
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.auth(w, r) {
		return
	}
	var cmd Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err == nil {
		select {
		case s.CommandChan <- cmd:
		default:
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleScores(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	scores, err := s.db.GetScores()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)

	filePath, err := s.db.GetScoreFilePath(id)
	if err != nil {
		http.Error(w, "Score not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	mgr, err := pdf.NewManager(filePath)
	if err != nil {
		http.Error(w, "PDF error", http.StatusInternalServerError)
		return
	}
	defer mgr.Close()

	img, err := mgr.GetPageImage(page)
	if err != nil {
		http.Error(w, "Page error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, img)
}
