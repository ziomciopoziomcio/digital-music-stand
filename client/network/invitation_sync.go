package network

import (
	"context"
	"fmt"
	"log"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
)

func SynchronizeInvitations(ctx context.Context, bandClient bandpb.BandServiceClient, dbMgr *localdb.DBManager) error {
	resp, err := bandClient.ListMyInvitations(ctx, &bandpb.ListMyInvitationsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list band invitations: %w", err)
	}

	for _, inv := range resp.GetInvitations() {
		notifID := fmt.Sprintf("inv_%d", inv.Id)
		refID := fmt.Sprintf("%d", inv.Id)
		title := "New Band Invitation"
		body := fmt.Sprintf("You have been invited to join '%s'.", inv.BandName)

		if err := dbMgr.SyncNotificationFromServer(notifID, "band_invite", refID, title, body); err != nil {
			log.Printf("Failed to sync band invitation %d: %v", inv.Id, err)
		}
	}

	return nil
}

func SynchronizeConcertInvitations(ctx context.Context, concertClient concertpb.ConcertServiceClient, dbMgr *localdb.DBManager) error {
	resp, err := concertClient.ListConcertInvitations(ctx, &concertpb.ListConcertInvitationsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list concert invitations: %w", err)
	}

	for _, inv := range resp.GetInvitations() {
		notifID := fmt.Sprintf("cinv_%d", inv.Id)
		refID := fmt.Sprintf("%d", inv.Id)
		title := "Concert Shared With You"
		body := fmt.Sprintf("'%s' was shared with you by %s.", inv.ConcertName, inv.OwnerEmail)

		if err := dbMgr.SyncNotificationFromServer(notifID, "concert_invite", refID, title, body); err != nil {
			log.Printf("Failed to sync concert invitation %d: %v", inv.Id, err)
		}
	}

	return nil
}

func SynchronizeScoreInvitations(ctx context.Context, scoreClient scorepb.ScoreServiceClient, dbMgr *localdb.DBManager) error {
	resp, err := scoreClient.ListScoreInvitations(ctx, &scorepb.ListScoreInvitationsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list score invitations: %w", err)
	}

	for _, inv := range resp.GetInvitations() {
		notifID := fmt.Sprintf("sinv_%d", inv.Id)
		refID := fmt.Sprintf("%d", inv.Id)
		title := "Score Shared With You"
		body := fmt.Sprintf("Score '%s' was shared with you by %s.", inv.ScoreName, inv.OwnerEmail)

		if err := dbMgr.SyncNotificationFromServer(notifID, "score_invite", refID, title, body); err != nil {
			log.Printf("Failed to sync score invitation %d: %v", inv.Id, err)
		}
	}

	return nil
}
