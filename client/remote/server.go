package remote

import (
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
	"github.com/ziomciopoziomcio/digital-music-stand/client/webserver"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
	"google.golang.org/grpc"
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

type PeerInfo struct {
	IP        string
	Port      int
	ConcertID string
	LastSeen  time.Time
}

type Server struct {
	syncpb.UnimplementedLiveSyncServiceServer
	mu          sync.RWMutex
	PIN         string
	State       ConcertState
	CommandChan chan Command
	server      *http.Server
	port        int
	db          *localdb.DBManager

	grpcServer   *grpc.Server
	grpcPort     int
	localStreams map[string]map[syncpb.LiveSyncService_SyncConcertStreamServer]bool
	peers        map[string]PeerInfo

	isLeading bool
	activeCID string
}

func NewServer(port int, db *localdb.DBManager) *Server {
	rand.Seed(time.Now().UnixNano())
	return &Server{
		PIN:          fmt.Sprintf("%04d", rand.Intn(10000)),
		CommandChan:  make(chan Command, 20),
		port:         port,
		db:           db,
		grpcPort:     50052,
		localStreams: make(map[string]map[syncpb.LiveSyncService_SyncConcertStreamServer]bool),
		peers:        make(map[string]PeerInfo),
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

	lis, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", s.grpcPort))
	if err == nil {
		s.grpcServer = grpc.NewServer()
		syncpb.RegisterLiveSyncServiceServer(s.grpcServer, s)
		go s.grpcServer.Serve(lis)
	}

	go s.p2pBroadcastLoop()
	go s.p2pListenLoop()
}

func (s *Server) SetLeading(concertID string, leading bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isLeading = leading
	s.activeCID = concertID
}

func (s *Server) p2pBroadcastLoop() {
	addr, _ := net.ResolveUDPAddr("udp4", "255.255.255.255:8091")
	conn, _ := net.DialUDP("udp4", nil, addr)
	if conn == nil {
		return
	}
	defer conn.Close()

	for {
		s.mu.RLock()
		leading := s.isLeading
		cid := s.activeCID
		s.mu.RUnlock()

		if leading && cid != "" {
			ip := webserver.GetLocalIP()
			msg := fmt.Sprintf("DMS_P2P|%s|%s|%d", cid, ip, s.grpcPort)
			conn.Write([]byte(msg))
		}
		time.Sleep(2 * time.Second)
	}
}

func (s *Server) p2pListenLoop() {
	addr, _ := net.ResolveUDPAddr("udp4", ":8091")
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return
	}
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err == nil {
			parts := strings.Split(string(buf[:n]), "|")
			if len(parts) == 4 && parts[0] == "DMS_P2P" {
				cid := parts[1]
				ip := parts[2]
				port, _ := strconv.Atoi(parts[3])

				if ip == webserver.GetLocalIP() {
					continue
				}

				s.mu.Lock()
				s.peers[cid] = PeerInfo{
					IP:        ip,
					Port:      port,
					ConcertID: cid,
					LastSeen:  time.Now(),
				}
				s.mu.Unlock()
			}
		}
	}
}

func (s *Server) GetDiscoveredPeers() []PeerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []PeerInfo
	now := time.Now()
	for _, p := range s.peers {
		if now.Sub(p.LastSeen) < 10*time.Second {
			list = append(list, p)
		}
	}
	return list
}

func (s *Server) GetPeerForConcert(concertID string) (PeerInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.peers[concertID]
	if ok && time.Since(p.LastSeen) < 10*time.Second {
		return p, true
	}
	return PeerInfo{}, false
}

func (s *Server) SyncConcertStream(stream syncpb.LiveSyncService_SyncConcertStreamServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	cid := req.GetConcertId()
	s.mu.Lock()
	if s.localStreams[cid] == nil {
		s.localStreams[cid] = make(map[syncpb.LiveSyncService_SyncConcertStreamServer]bool)
	}
	s.localStreams[cid][stream] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if s.localStreams[cid] != nil {
			delete(s.localStreams[cid], stream)
		}
		s.mu.Unlock()
	}()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		resp := &syncpb.SyncResponse{
			ConcertId:    req.GetConcertId(),
			Action:       req.GetAction(),
			PageNumber:   req.GetPageNumber(),
			ItemIndex:    req.GetItemIndex(),
			TimerSeconds: req.GetTimerSeconds(),
			IsAccent:     req.GetIsAccent(),
			IsLeader:     req.GetIsLeader(),
			TimestampMs:  time.Now().UnixMilli(),
			Payload:      req.GetPayload(),
		}

		s.mu.RLock()
		streams := s.localStreams[cid]
		s.mu.RUnlock()

		for st := range streams {
			if st != stream {
				_ = st.Send(resp)
			}
		}
	}
}

func (s *Server) BroadcastLocal(req *syncpb.SyncRequest) {
	cid := req.GetConcertId()
	resp := &syncpb.SyncResponse{
		ConcertId:    req.GetConcertId(),
		Action:       req.GetAction(),
		PageNumber:   req.GetPageNumber(),
		ItemIndex:    req.GetItemIndex(),
		TimerSeconds: req.GetTimerSeconds(),
		IsAccent:     req.GetIsAccent(),
		IsLeader:     req.GetIsLeader(),
		TimestampMs:  time.Now().UnixMilli(),
		Payload:      req.GetPayload(),
	}

	s.mu.RLock()
	streams := s.localStreams[cid]
	s.mu.RUnlock()

	for st := range streams {
		_ = st.Send(resp)
	}
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
		w.WriteHeader(http.StatusOK)
	}
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
