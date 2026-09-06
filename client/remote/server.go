package remote

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	mu          sync.Mutex
	PIN         string
	CommandChan chan string
	server      *http.Server
	port        int
}

func NewServer(port int) *Server {
	rand.Seed(time.Now().UnixNano())
	pin := fmt.Sprintf("%04d", rand.Intn(10000))

	return &Server{
		PIN:         pin,
		CommandChan: make(chan string, 10),
		port:        port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/remote", s.handleCommand)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", s.port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("REMOTE SERVER ERROR: %v\n", err)
		}
	}()
	return nil
}

func (s *Server) Stop() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *Server) GetPIN() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.PIN
}

func (s *Server) RegeneratePIN() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PIN = fmt.Sprintf("%04d", rand.Intn(10000))
	return s.PIN
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PIN    string `json:"pin"`
		Action string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	valid := (s.PIN == req.PIN)
	s.mu.Unlock()

	if !valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	select {
	case s.CommandChan <- req.Action:
	default:
	}

	w.WriteHeader(http.StatusOK)
}
