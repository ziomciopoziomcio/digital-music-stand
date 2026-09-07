package remote

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type ConcertState struct {
	ConcertName    string `json:"concert_name"`
	CurrentItemIdx int    `json:"current_item_idx"`
	CurrentPage    int    `json:"current_page"`
	TotalPages     int    `json:"total_pages"`
	IsTimerRunning bool   `json:"is_timer_running"`
	CurrentPDFPath string `json:"-"`
}

type Command struct {
	Action string `json:"action"`
	Value  int    `json:"value"`
}

type Server struct {
	mu          sync.RWMutex
	PIN         string
	State       ConcertState
	CommandChan chan Command
	server      *http.Server
	port        int
}

func NewServer(port int) *Server {
	rand.Seed(time.Now().UnixNano())
	return &Server{
		PIN:         fmt.Sprintf("%04d", rand.Intn(10000)),
		CommandChan: make(chan Command, 10),
		port:        port,
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/command", s.handleCommand)
	mux.HandleFunc("/api/score", s.handleScore)

	s.server = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", s.port), Handler: mux}
	go s.server.ListenAndServe()
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

func (s *Server) handleScore(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	s.mu.RLock()
	path := s.State.CurrentPDFPath
	s.mu.RUnlock()

	if _, err := os.Stat(path); path == "" || err != nil {
		http.Error(w, "No active score", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, path)
}
