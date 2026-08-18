package network

import (
	"context"
	"log"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
)

func CreateAndSyncConcert(serverAddr, token string, name, location, startTime string, scores []localdb.Score, dbMgr *localdb.DBManager) error {
	if serverAddr == "" || token == "" {
		log.Println("Offline mode: Skipping cloud concert creation")
		return nil
	}

	conn, err := NewGRPCClient(serverAddr, token)
	if err != nil {
		log.Printf("Cloud sync failed (connection): %v", err)
		return nil
	}
	defer conn.Close()

	client := concertpb.NewConcertServiceClient(conn)
	ctx := context.Background()

	createResp, err := client.CreateConcert(ctx, &concertpb.CreateConcertRequest{
		Name:      name,
		Location:  &location,
		StartTime: &startTime,
	})
	if err != nil {
		log.Printf("Cloud sync failed (create concert): %v", err)
		return nil
	}

	concertID := createResp.GetId()

	for i, score := range scores {
		scoreID := score.ID
		_, err := client.AddConcertItem(ctx, &concertpb.AddConcertItemRequest{
			ConcertId: concertID,
			ScoreId:   &scoreID,
			Order:     uint32(i + 1),
		})
		if err != nil {
			log.Printf("Cloud sync failed (add item %s): %v", score.ID, err)
			return nil
		}
	}

	if err := SynchronizeConcerts(ctx, client, dbMgr); err != nil {
		log.Printf("Cloud sync failed (synchronize): %v", err)
	}

	return nil
}
