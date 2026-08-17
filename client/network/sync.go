package network

import (
	"context"
	"fmt"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
)

func SynchronizeConcerts(ctx context.Context, concertClient concertpb.ConcertServiceClient, dbMgr *localdb.DBManager) error {
	resp, err := concertClient.ListConcerts(ctx, &concertpb.ListConcertsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list remote concerts: %v", err)
	}

	localConcerts, err := dbMgr.GetConcerts()
	if err != nil {
		return fmt.Errorf("failed to get local concerts: %v", err)
	}

	localChecksums := make(map[int]string)
	for _, lc := range localConcerts {
		localChecksums[lc.ID] = lc.Checksum
	}

	for _, remoteConcert := range resp.GetConcerts() {
		concertID := int(remoteConcert.GetId())
		localChecksum, exists := localChecksums[concertID]

		if !exists || localChecksum != remoteConcert.Checksum {
			fmt.Printf("updating concert %d (%s)...\n", concertID, remoteConcert.GetName())

			setlistResp, err := concertClient.GetConcertSetlist(ctx, &concertpb.GetConcertSetlistRequest{
				ConcertId: remoteConcert.GetId(),
			})
			if err != nil {
				fmt.Printf("Setlist download error for concert %d: %v\n", concertID, err)
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
					ID:        int(item.GetId()),
					SortOrder: int(item.GetOrder()),
					ScoreID:   scoreID,
					BreakMin:  breakMin,
				})
			}

			err = dbMgr.SyncConcertFromServer(concertID, remoteConcert.GetName(), remoteConcert.GetChecksum(), remoteItems)
			if err != nil {
				fmt.Printf("Failed to saved sync concert %d: %v\n", concertID, err)
			}
		} else {
			fmt.Printf("Concert %d (%s) is up to date.\n", concertID, remoteConcert.Name)
		}
	}
	return nil
}
