package services

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
)

type BandService struct {
	bandpb.UnimplementedBandServiceServer
	db *gorm.DB
}

func NewBandService(db *gorm.DB) *BandService {
	return &BandService{db: db}
}

func (s *BandService) CreateBand(ctx context.Context, req *bandpb.CreateBandRequest) (*bandpb.CreateBandResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	if req.GetName() == "" {
		return nil, fmt.Errorf("band name cannot be empty")
	}

	band := models.Band{
		Name:      req.GetName(),
		ManagerID: userID,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&band).Error; err != nil {
			return err
		}

		member := models.BandMember{
			UserID: userID,
			BandID: band.ID,
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create band: %v", err)
	}

	return &bandpb.CreateBandResponse{
		Id:      uint32(band.ID),
		Message: "Band created successfully",
	}, nil
}

func (s *BandService) InviteMember(ctx context.Context, req *bandpb.InviteMemberRequest) (*bandpb.InviteMemberResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var band models.Band
	if err := s.db.Where("id = ? AND manager_id = ?", req.GetBandId(), userID).First(&band).Error; err != nil {
		return nil, fmt.Errorf("band not found or permission denied")
	}

	if req.GetInviteeEmail() == "" {
		return nil, fmt.Errorf("invitee email cannot be empty")
	}

	invite := models.BandInvitation{
		BandID:       uint(req.GetBandId()),
		InviteeEmail: req.GetInviteeEmail(),
		Status:       "pending",
	}

	if err := s.db.FirstOrCreate(&invite, models.BandInvitation{BandID: invite.BandID, InviteeEmail: invite.InviteeEmail}).Error; err != nil {
		return nil, fmt.Errorf("failed to create invitation: %v", err)
	}

	return &bandpb.InviteMemberResponse{
		Message: "Invitation sent successfully",
	}, nil
}

func (s *BandService) ListMyInvitations(ctx context.Context, req *bandpb.ListMyInvitationsRequest) (*bandpb.ListMyInvitationsResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	var invites []models.BandInvitation
	if err := s.db.Preload("Band").Where("invitee_email = ? AND status = ?", user.Email, "pending").Find(&invites).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch invitations: %v", err)
	}

	var response []*bandpb.Invitation
	for _, inv := range invites {
		response = append(response, &bandpb.Invitation{
			Id:       uint32(inv.ID),
			BandName: inv.Band.Name,
			Status:   inv.Status,
		})
	}

	return &bandpb.ListMyInvitationsResponse{
		Invitations: response,
	}, nil
}

func (s *BandService) RespondToInvitation(ctx context.Context, req *bandpb.RespondToInvitationRequest) (*bandpb.RespondToInvitationResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	var invite models.BandInvitation
	if err := s.db.Where("id = ? AND invitee_email = ? AND status = ?", req.GetInvitationId(), user.Email, "pending").First(&invite).Error; err != nil {
		return nil, fmt.Errorf("invitation not found or already processed")
	}

	status := "declined"
	if req.GetAccept() {
		status = "accepted"
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&invite).Update("status", status).Error; err != nil {
			return err
		}

		if req.GetAccept() {
			member := models.BandMember{
				UserID: userID,
				BandID: invite.BandID,
			}
			if err := tx.FirstOrCreate(&member, member).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to process invitation: %v", err)
	}

	return &bandpb.RespondToInvitationResponse{
		Message: "Invitation processed successfully",
	}, nil
}

func (s *BandService) ListMyBands(ctx context.Context, req *bandpb.ListMyBandsRequest) (*bandpb.ListMyBandsResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var members []models.BandMember
	if err := s.db.Preload("Band").Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch bands: %v", err)
	}

	var response []*bandpb.Band
	for _, m := range members {
		response = append(response, &bandpb.Band{
			Id:        uint32(m.Band.ID),
			Name:      m.Band.Name,
			IsManager: m.Band.ManagerID == userID,
		})
	}

	return &bandpb.ListMyBandsResponse{
		Bands: response,
	}, nil
}
