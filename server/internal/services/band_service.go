package services

import (
	"context"
	"fmt"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"gorm.io/gorm"
)

type BandService struct {
	bandpb.UnimplementedBandServiceServer
	db *gorm.DB
}

func NewBandService(db *gorm.DB) *BandService {
	return &BandService{
		db: db,
	}
}

func (s *BandService) CreateBand(ctx context.Context, req *bandpb.CreateBandRequest) (*bandpb.CreateBandResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id from context: %v", err)
	}

	if req.GetName() == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	var newBand models.Band
	err = s.db.Transaction(func(tx *gorm.DB) error {
		newBand = models.Band{
			Name:      req.GetName(),
			ManagerID: userID,
		}

		if err := tx.Create(&newBand).Error; err != nil {
			return fmt.Errorf("failed to create band: %v", err)
		}

		firstMember := models.BandMember{
			BandID: newBand.ID,
			UserID: userID,
		}
		if err := tx.Create(&firstMember).Error; err != nil {
			return fmt.Errorf("failed to add manager as first member: %v", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create band: %v", err)
	}

	return &bandpb.CreateBandResponse{
		Id:      uint32(newBand.ID),
		Message: "Band created successfully",
	}, nil
}

func (s *BandService) AddBandMember(ctx context.Context, req *bandpb.AddBandMemberRequest) (*bandpb.AddBandMemberResponse, error) {
	if req.GetBandId() == 0 || req.GetUserId() == 0 {
		return nil, fmt.Errorf("missing required fields")
	}

	authUserID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id from context: %v", err)
	}

	var band models.Band
	if err := s.db.Select("manager_id").First(&band, req.GetBandId()).Error; err != nil {
		return nil, fmt.Errorf("failed to find band: %v", err)
	}

	if band.ManagerID != authUserID {
		return nil, fmt.Errorf("only the band manager can add members")
	}

	newMember := models.BandMember{
		BandID: uint(req.GetBandId()),
		UserID: uint(req.GetUserId()),
	}

	if err := s.db.Create(&newMember).Error; err != nil {
		return nil, fmt.Errorf("failed to add band member: %v", err)
	}

	return &bandpb.AddBandMemberResponse{
		Message: "Band member added successfully",
	}, nil
}

func (s *BandService) ListBandMembers(ctx context.Context, req *bandpb.ListBandMembersRequest) (*bandpb.ListBandMembersResponse, error) {
	if req.GetBandId() == 0 { // todo: Check if the user is a member of the band or the manager
		return nil, fmt.Errorf("missing required fields")
	}

	authUserID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id from context: %v", err)
	}

	var count int64

	if err := s.db.Model(&models.BandMember{}).
		Where("band_id = ? AND user_id = ?", req.GetBandId(), authUserID).
		Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check if user is a member of the band: %v", err)
	}

	if count == 0 {
		return nil, fmt.Errorf("user is not a member of the band")
	}

	type MemberResult struct {
		ID      uint
		Name    string
		Surname string
	}
	var results []MemberResult

	err := s.db.Table("band_members").
		Select("users.id, users.name, users.surname").
		Joins("JOIN users ON band_members.user_id = users.id").
		Where("band_members.band_id = ?", req.GetBandId()).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list band members: %v", err)
	}

	var grpcMembers []*bandpb.BandMember
	for _, res := range results {
		grpcMembers = append(grpcMembers, &bandpb.BandMember{
			Id:      uint32(res.ID),
			Name:    res.Name,
			Surname: res.Surname,
		})
	}

	return &bandpb.ListBandMembersResponse{
		Members: grpcMembers,
	}, nil
}
