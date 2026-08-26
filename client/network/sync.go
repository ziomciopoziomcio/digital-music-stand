package network

import (
	"context"
	"fmt"
	"log"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
)

func SynchronizeConcerts(ctx context.Context, concertClient concertpb.ConcertServiceClient, dbMgr *localdb.DBManager) error {
	deletedIDs, err := dbMgr.GetDeletedConcertIDs()
	if err == nil {
		for _, id := range deletedIDs {
			log.Printf("Pushing deletion for concert %s to cloud...", id)
			_, err := concertClient.DeleteConcert(ctx, &concertpb.DeleteConcertRequest{Id: id})
			if err != nil {
				log.Printf("Failed to delete concert %s on server: %v", id, err)
			} else {
				_ = dbMgr.HardDeleteConcert(id)
			}
		}
	}

	remoteResp, err := concertClient.ListMyConcerts(ctx, &concertpb.ListMyConcertsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list remote concerts: %w", err)
	}

	remoteMap := make(map[string]*concertpb.Concert)
	for _, rc := range remoteResp.GetConcerts() {
		remoteMap[rc.Id] = rc
	}

	localConcerts, err := dbMgr.GetConcerts()
	if err != nil {
		return fmt.Errorf("failed to get local concerts: %w", err)
	}

	for _, lc := range localConcerts {
		if lc.Checksum == "" {
			if _, existsOnServer := remoteMap[lc.ID]; existsOnServer {
				log.Printf("Updating remote concert %s (%s)...", lc.Name, lc.ID)
				_, err := concertClient.UpdateConcert(ctx, &concertpb.UpdateConcertRequest{
					Id:        lc.ID,
					Name:      lc.Name,
					Location:  &lc.Location,
					StartTime: &lc.StartTime,
				})
				if err != nil {
					log.Printf("Failed to update concert %s on server: %v", lc.Name, err)
					continue
				}

				for _, item := range lc.Items {
					req := &concertpb.AddConcertItemRequest{
						ConcertId: lc.ID,
						Order:     uint32(item.SortOrder),
					}
					if item.ScoreID != nil {
						req.ScoreId = item.ScoreID
					} else if item.BreakMin != nil {
						breakMin := uint32(*item.BreakMin)
						req.BreakMin = &breakMin
					}
					_, _ = concertClient.AddConcertItem(ctx, req)
				}
			} else {
				log.Printf("Pushing new concert %s (%s) to cloud...", lc.Name, lc.ID)
				_, err := concertClient.CreateConcert(ctx, &concertpb.CreateConcertRequest{
					Id:        &lc.ID,
					Name:      lc.Name,
					Location:  &lc.Location,
					StartTime: &lc.StartTime,
				})
				if err != nil {
					log.Printf("Failed to push concert %s to cloud: %v", lc.Name, err)
					continue
				}

				for _, item := range lc.Items {
					req := &concertpb.AddConcertItemRequest{
						ConcertId: lc.ID,
						Order:     uint32(item.SortOrder),
					}
					if item.ScoreID != nil {
						req.ScoreId = item.ScoreID
					} else if item.BreakMin != nil {
						breakMin := uint32(*item.BreakMin)
						req.BreakMin = &breakMin
					}
					_, _ = concertClient.AddConcertItem(ctx, req)
				}
			}
		}
	}

	remoteResp, err = concertClient.ListMyConcerts(ctx, &concertpb.ListMyConcertsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list remote concerts after push: %w", err)
	}

	remoteMap = make(map[string]*concertpb.Concert)
	for _, rc := range remoteResp.GetConcerts() {
		remoteMap[rc.Id] = rc
	}

	localConcertsMap := make(map[string]localdb.Concert)
	localConcerts, _ = dbMgr.GetConcerts()
	for _, lc := range localConcerts {
		localConcertsMap[lc.ID] = lc
	}

	for _, remoteConcert := range remoteResp.GetConcerts() {
		localConcert, exists := localConcertsMap[remoteConcert.Id]

		if !exists || localConcert.Checksum != remoteConcert.Checksum {
			log.Printf("Updating concert %s (%s) due to checksum mismatch", remoteConcert.Id, remoteConcert.Name)

			var remoteItems []localdb.ConcertItem
			for _, item := range remoteConcert.GetItems() {
				var scoreID *string
				if item.ScoreId != nil && *item.ScoreId != "" {
					scoreID = item.ScoreId
				}
				var breakMin *int
				if item.BreakMin != nil {
					val := int(*item.BreakMin)
					breakMin = &val
				}

				remoteItems = append(remoteItems, localdb.ConcertItem{
					ID:        item.Id,
					SortOrder: int(item.Order),
					ScoreID:   scoreID,
					BreakMin:  breakMin,
				})
			}

			loc := remoteConcert.GetLocation()
			startTime := remoteConcert.GetStartTime()

			err = dbMgr.SyncConcertFromServer(remoteConcert.Id, remoteConcert.Name, loc, startTime, remoteConcert.Checksum, remoteItems, remoteConcert.IsOwner, remoteConcert.CanEdit)
			if err != nil {
				log.Printf("Failed to save concert %s to local database: %v", remoteConcert.Id, err)
			}
		}
	}

	for _, lc := range localConcerts {
		if lc.Checksum != "" {
			if _, existsOnServer := remoteMap[lc.ID]; !existsOnServer {
				log.Printf("Concert %s (%s) was deleted on server. Removing locally...", lc.Name, lc.ID)
				_ = dbMgr.HardDeleteConcert(lc.ID)
			}
		}
	}

	return nil
}
