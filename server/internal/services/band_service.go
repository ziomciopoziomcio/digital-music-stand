package services

import (
	"context"
	"fmt"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/bandpb"
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
	if req.GetName() == "" || req.GetManagerId() == 0 { // todo: ManagerID should be extracted from JWT token
		return nil, fmt.Errorf("missing required fields")
	}

	var newBand models.Band
	err := s.db.Transaction(func(tx *gorm.DB) error {
		newBand = models.Band{
			Name:      req.GetName(),
			ManagerID: uint(req.GetManagerId()),
		}

		if err := tx.Create(&newBand).Error; err != nil {
			return fmt.Errorf("failed to create band: %v", err)
		}

		firstMember := models.BandMember{
			BandID: newBand.ID,
			UserID: uint(req.GetManagerId()),
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

	//todo: check if the user is manager of the band

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
