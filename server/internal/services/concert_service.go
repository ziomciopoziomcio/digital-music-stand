package services

import (
	"context"
	"fmt"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"gorm.io/gorm"
)

type ConcertService struct {
	concertpb.UnimplementedConcertServiceServer
	db *gorm.DB
}

func NewConcertService(db *gorm.DB) *ConcertService {
	return &ConcertService{
		db: db,
	}
}

func (s *ConcertService) CreateConcert(ctx context.Context, req *concertpb.CreateConcertRequest) (*concertpb.CreateConcertResponse, error) {
	if req.GetName() == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	if req.BandId == nil && req.UserId == nil {
		return nil, fmt.Errorf("either BandId or UserId must be provided")
	}
	if req.BandId != nil && req.UserId != nil {
		return nil, fmt.Errorf("only one of BandId or UserId can be provided")
	}

	var bandID, userID *uint
	if req.BandId != nil {
		id := uint(*req.BandId)
		bandID = &id
	}
	if req.UserId != nil {
		id := uint(*req.UserId)
		userID = &id
	}

	newConcert := models.Concert{
		Name:   req.GetName(),
		BandID: bandID,
		UserID: userID,
	}

	if err := s.db.Create(&newConcert).Error; err != nil {
		return nil, fmt.Errorf("failed to create concert: %v", err)
	}

	return &concertpb.CreateConcertResponse{
		Id:      uint32(newConcert.ID),
		Message: "Concert created successfully",
	}, nil
}

func (s *ConcertService) AddConcertItem(ctx context.Context, req *concertpb.AddConcertItemRequest) (*concertpb.AddConcertItemResponse, error) {
	if req.GetConcertId() == 0 { // todo: check for permission
		return nil, fmt.Errorf("missing required fields")
	}
	if req.ScoreId == nil && req.BreakMin == nil {
		return nil, fmt.Errorf("either ScoreId or BreakMin must be provided")
	}
	if req.ScoreId != nil && req.BreakMin != nil {
		return nil, fmt.Errorf("only one of ScoreId or BreakMin can be provided")
	}

	var scoreID *uint
	var breakMin *int
	if req.ScoreId != nil {
		id := uint(*req.ScoreId)
		scoreID = &id
	}
	if req.BreakMin != nil {
		val := int(*req.BreakMin)
		breakMin = &val
	}

	newItem := models.ConcertItem{
		ConcertID: uint(req.GetConcertId()),
		ScoreID:   scoreID,
		BreakMin:  breakMin,
		SortOrder: int(req.GetOrder()),
	}

	if err := s.db.Create(&newItem).Error; err != nil {
		return nil, fmt.Errorf("failed to add concert item: %v", err)
	}

	return &concertpb.AddConcertItemResponse{
		Id:      uint32(newItem.ID),
		Message: "Concert item added successfully",
	}, nil
}

func (s *ConcertService) GetConcertSetlist(ctx context.Context, req *concertpb.GetConcertSetlistRequest) (*concertpb.GetConcertSetlistResponse, error) {
	if req.GetConcertId() == 0 { // todo: check for permission
		return nil, fmt.Errorf("missing required fields")
	}

	var items []models.ConcertItem

	if err := s.db.Preload("Score").Where("concert_id = ?", req.GetConcertId()).Order("sort_order asc").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to get concert setlist: %v", err)
	}

	var grpcItems []*concertpb.ConcertItem
	for _, item := range items {
		grpcItem := &concertpb.ConcertItem{
			Id:    uint32(item.ID),
			Order: uint32(item.SortOrder),
		}

		if item.ScoreID != nil {
			scoreID := uint32(*item.ScoreID)
			grpcItem.ScoreId = &scoreID

			scoreName := item.Score.Name
			filePath := item.Score.FilePath
			grpcItem.ScoreName = &scoreName
			grpcItem.FilePath = &filePath
		} else if item.BreakMin != nil {
			breakMin := uint32(*item.BreakMin)
			grpcItem.BreakMin = &breakMin
		}

		grpcItems = append(grpcItems, grpcItem)
	}
	return &concertpb.GetConcertSetlistResponse{
		Items: grpcItems,
	}, nil
}
