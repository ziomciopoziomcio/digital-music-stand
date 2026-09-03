package services

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
)

type streamWrapper struct {
	stream syncpb.LiveSyncService_SyncConcertStreamServer
	mu     sync.Mutex
}

type LiveSyncService struct {
	syncpb.UnimplementedLiveSyncServiceServer
	mu          sync.RWMutex
	connections map[string]map[string]*streamWrapper
}

func NewLiveSyncService() *LiveSyncService {
	return &LiveSyncService{
		connections: make(map[string]map[string]*streamWrapper),
	}
}

func (s *LiveSyncService) SyncConcertStream(stream syncpb.LiveSyncService_SyncConcertStreamServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	concertID := req.GetConcertId()
	authUserID, err := auth.GetUserIDFromContext(stream.Context())
	if err != nil {
		return err
	}

	connID := fmt.Sprintf("%s-%d-%d", concertID, authUserID, time.Now().UnixNano())
	wrapper := &streamWrapper{stream: stream}

	s.mu.Lock()
	if s.connections[concertID] == nil {
		s.connections[concertID] = make(map[string]*streamWrapper)
	}
	s.connections[concertID][connID] = wrapper
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.connections[concertID], connID)
		if len(s.connections[concertID]) == 0 {
			delete(s.connections, concertID)
		}
		s.mu.Unlock()
	}()

	s.broadcast(concertID, uint32(authUserID), req)

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		s.broadcast(concertID, uint32(authUserID), req)
	}
}

func (s *LiveSyncService) broadcast(concertID string, senderID uint32, req *syncpb.SyncRequest) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	streams, ok := s.connections[concertID]
	if !ok {
		return
	}

	resp := &syncpb.SyncResponse{
		ConcertId:    req.GetConcertId(),
		SenderId:     senderID,
		Action:       req.GetAction(),
		PageNumber:   req.GetPageNumber(),
		ItemIndex:    req.GetItemIndex(),
		TimerSeconds: req.GetTimerSeconds(),
		IsAccent:     req.GetIsAccent(),
		IsLeader:     req.GetIsLeader(),
		TimestampMs:  time.Now().UnixMilli(),
	}

	for _, w := range streams {
		w.mu.Lock()
		_ = w.stream.Send(resp)
		w.mu.Unlock()
	}
}
