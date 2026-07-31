package services

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/syncpb"
)

type LiveSyncService struct {
	syncpb.UnimplementedLiveSyncServiceServer

	mu          sync.RWMutex
	connections map[uint32]map[string]syncpb.LiveSyncService_SyncConcertStreamServer
}

func NewLiveSyncService() *LiveSyncService {
	return &LiveSyncService{
		connections: make(map[uint32]map[string]syncpb.LiveSyncService_SyncConcertStreamServer),
	}
}

func (s *LiveSyncService) SyncConcertStream(stream syncpb.LiveSyncService_SyncConcertStreamServer) error {
	var concertID uint32
	var userID uint32

	req, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		log.Printf("error receiving concert from stream: %v", err)
		return err
	}

	concertID = req.ConcertId
	userID = req.UserId
	connID := fmt.Sprintf("concert-%d-user-%d", concertID, userID)

	s.mu.Lock()
	if s.connections[concertID] == nil {
		s.connections[concertID] = make(map[string]syncpb.LiveSyncService_SyncConcertStreamServer)
	}
	s.connections[concertID][connID] = stream
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.connections[concertID], connID)
		if len(s.connections[concertID]) == 0 {
			delete(s.connections, concertID)
		}
		s.mu.Unlock()
		log.Printf("disconnected from concert %d, user %s", concertID, userID)
	}()

	log.Printf("connected to concert %d, user %s", concertID, userID)

	s.broadcast(concertID, req)

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Printf("error receiving from stream: %v", err)
			return err
		}
		s.broadcast(concertID, req)
	}
}

func (s *LiveSyncService) broadcast(concertID uint32, req *syncpb.SyncRequest) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	streams, ok := s.connections[concertID]
	if !ok {
		return
	}

	resp := &syncpb.SyncResponse{
		ConcertId:     req.ConcertId,
		SenderId:      req.UserId,
		Action:        req.Action,
		PageNumber:    req.PageNumber,
		MeasureNumber: req.MeasureNumber,
		TimestampMs:   time.Now().UnixMilli(),
	}

	for id, stream := range streams {
		if err := stream.Send(resp); err != nil {
			log.Printf("error sending to stream %s: %v", id, err)
		}
	}
}
