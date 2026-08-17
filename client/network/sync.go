package network

import (
	"context"
	"fmt"
	"log"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
)

func SynchronizeConcerts(ctx context.Context, concertClient concertpb.ConcertServiceClient, dbMgr *localdb.DBManager) error {
	remoteResp, err := concertClient.ListConcerts(ctx, &concertpb.ListConcertsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list remote concerts: %w", err)
	}

	remoteMap := make(map[int]*concertpb.ConcertSummary)
	for _, rc := range remoteResp.GetConcerts() {
		remoteMap[int(rc.Id)] = rc
	}

	localConcerts, err := dbMgr.GetConcerts()
	if err != nil {
		return fmt.Errorf("failed to get local concerts: %w", err)
	}

	for _, lc := range localConcerts {
		if lc.Checksum == "" {
			log.Printf("Pushing local concert %s to cloud...", lc.Name)
			createResp, err := concertClient.CreateConcert(ctx, &concertpb.CreateConcertRequest{
				Name:      lc.Name,
				Location:  &lc.Location,
				StartTime: &lc.StartTime,
			})
			if err != nil {
				log.Printf("Failed to push concert %s to cloud: %v", lc.Name, err)
				continue
			}

			remoteID := createResp.GetId()
			for _, item := range lc.Items {
				req := &concertpb.AddConcertItemRequest{
					ConcertId: remoteID,
					Order:     uint32(item.SortOrder),
				}
				if item.ScoreID != nil {
					scoreID := uint32(*item.ScoreID)
					req.ScoreId = &scoreID
				} else if item.BreakMin != nil {
					breakMin := uint32(*item.BreakMin)
					req.BreakMin = &breakMin
				}

				if _, err := concertClient.AddConcertItem(ctx, req); err != nil {
					log.Printf("Failed to push concert item to cloud: %v", err)
				}
			}
		}
	}

	remoteResp, err = concertClient.ListConcerts(ctx, &concertpb.ListConcertsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list remote concerts after push: %w", err)
	}

	localConcertsMap := make(map[int]localdb.Concert)
	for _, lc := range localConcerts {
		localConcertsMap[lc.ID] = lc
	}

	for _, remoteConcert := range remoteResp.GetConcerts() {
		concertID := int(remoteConcert.Id)
		localConcert, exists := localConcertsMap[concertID]

		if !exists || localConcert.Checksum != remoteConcert.Checksum {
			log.Printf("Updating concert %d (%s) due to checksum mismatch", concertID, remoteConcert.Name)

			setlistResp, err := concertClient.GetConcertSetlist(ctx, &concertpb.GetConcertSetlistRequest{
				ConcertId: remoteConcert.Id,
			})
			if err != nil {
				log.Printf("Failed to fetch setlist for concert %d: %v", concertID, err)
				continue
			}

			var remoteItems []localdb.ConcertItem
			for _, item := range setlistResp.GetItems() {
				var scoreID *int
				if item.ScoreId != nil {
					id := int(*item.ScoreId)
					scoreID = &id
				}
				var breakMin *int
				if item.BreakMin != nil {
					val := int(*item.BreakMin)
					breakMin = &val
				}

				remoteItems = append(remoteItems, localdb.ConcertItem{
					ID:        int(item.Id),
					SortOrder: int(item.Order),
					ScoreID:   scoreID,
					BreakMin:  breakMin,
				})
			}

			loc := ""
			if remoteConcert.Location != nil {
				loc = *remoteConcert.Location
			}
			startTime := ""
			if remoteConcert.StartTime != nil {
				startTime = *remoteConcert.StartTime
			}

			err = dbMgr.SyncConcertFromServer(concertID, remoteConcert.Name, loc, startTime, remoteConcert.Checksum, remoteItems)
			if err != nil {
				log.Printf("Failed to save concert %d to local database: %v", concertID, err)
			}
		} else {
			log.Printf("Concert %d (%s) is up to date", concertID, remoteConcert.Name)
		}
	}

	return nil
}
