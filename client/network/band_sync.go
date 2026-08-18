package network

import (
	"context"
	"fmt"
	"log"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
)

func SynchronizeInvitations(ctx context.Context, bandClient bandpb.BandServiceClient, dbMgr *localdb.DBManager) error {
	resp, err := bandClient.ListMyInvitations(ctx, &bandpb.ListMyInvitationsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list invitations: %w", err)
	}

	for _, inv := range resp.GetInvitations() {
		notifID := fmt.Sprintf("inv_%d", inv.Id)
		refID := fmt.Sprintf("%d", inv.Id)
		title := "New Band Invitation"
		body := fmt.Sprintf("You have been invited to join '%s'.", inv.BandName)

		err := dbMgr.SyncNotificationFromServer(notifID, "band_invite", refID, title, body)
		if err != nil {
			log.Printf("Failed to sync invitation %d: %v", inv.Id, err)
		}
	}

	return nil
}
