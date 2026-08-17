package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
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

func (s *ConcertService) calculateConcertChecksum(concertID uint) (string, error) {
	var concert models.Concert
	if err := s.db.First(&concert, concertID).Error; err != nil {
		return "", err
	}

	var items []models.ConcertItem
	if err := s.db.Where("concert_id = ?", concertID).Order("sort_order asc").Find(&items).Error; err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write([]byte(concert.Name))
	h.Write([]byte(concert.Location))
	h.Write([]byte(concert.StartTime))

	for _, item := range items {
		scoreStr := "nil"
		if item.ScoreID != nil {
			scoreStr = fmt.Sprintf("%d", *item.ScoreID)
		}
		breakStr := "nil"
		if item.BreakMin != nil {
			breakStr = fmt.Sprintf("%d", *item.BreakMin)
		}
		h.Write([]byte(fmt.Sprintf("|%d:%s:%s", item.SortOrder, scoreStr, breakStr)))
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *ConcertService) updateConcertChecksum(concertID uint) error {
	sum, err := s.calculateConcertChecksum(concertID)
	if err != nil {
		return err
	}
	return s.db.Model(&models.Concert{}).Where("id = ?", concertID).Update("checksum", sum).Error
}

func (s *ConcertService) CreateConcert(ctx context.Context, req *concertpb.CreateConcertRequest) (*concertpb.CreateConcertResponse, error) {
	authUserID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetName() == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	var bandID, userID *uint
	if req.BandId != nil {
		id := uint(*req.BandId)
		bandID = &id
	} else {
		userID = &authUserID
	}

	loc := ""
	if req.Location != nil {
		loc = *req.Location
	}
	startTime := ""
	if req.StartTime != nil {
		startTime = *req.StartTime
	}

	newConcert := models.Concert{
		Name:      req.GetName(),
		Location:  loc,
		StartTime: startTime,
		BandID:    bandID,
		UserID:    userID,
		Checksum:  "initial",
	}

	if err := s.db.Create(&newConcert).Error; err != nil {
		return nil, fmt.Errorf("failed to create concert: %v", err)
	}

	_ = s.updateConcertChecksum(newConcert.ID)

	return &concertpb.CreateConcertResponse{
		Id:      uint32(newConcert.ID),
		Message: "Concert created successfully",
	}, nil
}

func (s *ConcertService) AddConcertItem(ctx context.Context, req *concertpb.AddConcertItemRequest) (*concertpb.AddConcertItemResponse, error) {
	if req.GetConcertId() == 0 {
		return nil, fmt.Errorf("missing required fields")
	}

	if err := s.checkConcertPermission(ctx, uint(req.GetConcertId())); err != nil {
		return nil, err
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

	_ = s.updateConcertChecksum(uint(req.GetConcertId()))

	return &concertpb.AddConcertItemResponse{
		Id:      uint32(newItem.ID),
		Message: "Concert item added successfully",
	}, nil
}

func (s *ConcertService) GetConcertSetlist(ctx context.Context, req *concertpb.GetConcertSetlistRequest) (*concertpb.GetConcertSetlistResponse, error) {
	if req.GetConcertId() == 0 {
		return nil, fmt.Errorf("missing required fields")
	}
	if err := s.checkConcertPermission(ctx, uint(req.GetConcertId())); err != nil {
		return nil, err
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

func (s *ConcertService) ListConcerts(ctx context.Context, req *concertpb.ListConcertsRequest) (*concertpb.ListConcertsResponse, error) {
	authUserID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var concerts []models.Concert
	query := s.db.Where("user_id = ?", authUserID)
	if req.BandId != nil {
		query = s.db.Where("band_id = ?", *req.BandId)
	}

	if err := query.Find(&concerts).Error; err != nil {
		return nil, fmt.Errorf("failed to list concerts: %v", err)
	}

	var summaries []*concertpb.ConcertSummary
	for _, c := range concerts {
		loc := c.Location
		start := c.StartTime
		summaries = append(summaries, &concertpb.ConcertSummary{
			Id:        uint32(c.ID),
			Name:      c.Name,
			Checksum:  c.Checksum,
			Location:  &loc,
			StartTime: &start,
		})
	}

	return &concertpb.ListConcertsResponse{
		Concerts: summaries,
	}, nil
}

func (s *ConcertService) checkConcertPermission(ctx context.Context, concertID uint) error {
	authUserID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	var concert models.Concert
	if err := s.db.Select("user_id, band_id").First(&concert, concertID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("concert not found")
		}
		return fmt.Errorf("failed to find concert: %v", err)
	}

	if concert.UserID != nil {
		if *concert.UserID != authUserID {
			return fmt.Errorf("user does not have permission to modify this concert")
		}
		return nil
	}
	if concert.BandID != nil {
		var count int64
		if err := s.db.Model(&models.BandMember{}).
			Where("band_id = ? AND user_id = ?", *concert.BandID, authUserID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check band membership: %v", err)
		}
		if count == 0 {
			return fmt.Errorf("user does not have permission to modify this concert")
		}
		return nil
	}
	return fmt.Errorf("invalid concert data: neither user_id nor band_id is set")
}
