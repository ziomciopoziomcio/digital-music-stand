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

func (s *ConcertService) calculateConcertChecksum(concertID string) (string, error) {
	var concert models.Concert
	if err := s.db.First(&concert, "id = ?", concertID).Error; err != nil {
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
			scoreStr = *item.ScoreID
		}
		breakStr := "nil"
		if item.BreakMin != nil {
			breakStr = fmt.Sprintf("%d", *item.BreakMin)
		}
		h.Write([]byte(fmt.Sprintf("|%d:%s:%s", item.SortOrder, scoreStr, breakStr)))
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *ConcertService) updateConcertChecksum(concertID string) error {
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

	// Jeśli klient przysłał UUID (stworzył koncert offline), używamy go.
	// W przeciwnym razie GORM wygeneruje nowy za pomocą hooka BeforeCreate.
	if req.Id != nil && *req.Id != "" {
		newConcert.ID = *req.Id
	}

	if err := s.db.Create(&newConcert).Error; err != nil {
		return nil, fmt.Errorf("failed to create concert: %v", err)
	}

	_ = s.updateConcertChecksum(newConcert.ID)

	return &concertpb.CreateConcertResponse{
		Id:      newConcert.ID,
		Message: "Concert created successfully",
	}, nil
}

func (s *ConcertService) AddConcertItem(ctx context.Context, req *concertpb.AddConcertItemRequest) (*concertpb.AddConcertItemResponse, error) {
	if req.GetConcertId() == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	if err := s.checkConcertPermission(ctx, req.GetConcertId()); err != nil {
		return nil, err
	}

	if req.ScoreId == nil && req.BreakMin == nil {
		return nil, fmt.Errorf("either ScoreId or BreakMin must be provided")
	}
	if req.ScoreId != nil && req.BreakMin != nil {
		return nil, fmt.Errorf("only one of ScoreId or BreakMin can be provided")
	}

	var scoreID *string
	if req.ScoreId != nil && *req.ScoreId != "" {
		id := *req.ScoreId
		scoreID = &id
	}

	var breakMin *int
	if req.BreakMin != nil {
		val := int(*req.BreakMin)
		breakMin = &val
	}

	newItem := models.ConcertItem{
		ConcertID: req.GetConcertId(),
		ScoreID:   scoreID,
		BreakMin:  breakMin,
		SortOrder: int(req.GetOrder()),
	}

	if err := s.db.Create(&newItem).Error; err != nil {
		return nil, fmt.Errorf("failed to add concert item: %v", err)
	}

	_ = s.updateConcertChecksum(req.GetConcertId())

	return &concertpb.AddConcertItemResponse{
		Id:      newItem.ID,
		Message: "Concert item added successfully",
	}, nil
}

func (s *ConcertService) GetConcertSetlist(ctx context.Context, req *concertpb.GetConcertSetlistRequest) (*concertpb.GetConcertSetlistResponse, error) {
	if req.GetConcertId() == "" {
		return nil, fmt.Errorf("missing required fields")
	}
	if err := s.checkConcertPermission(ctx, req.GetConcertId()); err != nil {
		return nil, err
	}

	var items []models.ConcertItem
	if err := s.db.Preload("Score").Where("concert_id = ?", req.GetConcertId()).Order("sort_order asc").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to get concert setlist: %v", err)
	}

	var grpcItems []*concertpb.ConcertItem
	for _, item := range items {
		grpcItem := &concertpb.ConcertItem{
			Id:    item.ID,
			Order: uint32(item.SortOrder),
		}

		if item.ScoreID != nil {
			grpcItem.ScoreId = item.ScoreID

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
			Id:        c.ID,
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

func (s *ConcertService) checkConcertPermission(ctx context.Context, concertID string) error {
	authUserID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	var concert models.Concert
	if err := s.db.Select("user_id, band_id").First(&concert, "id = ?", concertID).Error; err != nil {
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

func (s *ConcertService) UpdateConcert(ctx context.Context, req *concertpb.UpdateConcertRequest) (*concertpb.UpdateConcertResponse, error) {
	if req.GetId() == "" {
		return nil, fmt.Errorf("missing concert id")
	}

	if err := s.checkConcertPermission(ctx, req.GetId()); err != nil {
		return nil, err
	}

	loc := ""
	if req.Location != nil {
		loc = *req.Location
	}
	startTime := ""
	if req.StartTime != nil {
		startTime = *req.StartTime
	}

	err := s.db.Model(&models.Concert{}).Where("id = ?", req.GetId()).Updates(map[string]interface{}{
		"name":       req.GetName(),
		"location":   loc,
		"start_time": startTime,
	}).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update concert: %v", err)
	}

	if err := s.db.Where("concert_id = ?", req.GetId()).Delete(&models.ConcertItem{}).Error; err != nil {
		return nil, fmt.Errorf("failed to clear old concert items: %v", err)
	}

	_ = s.updateConcertChecksum(req.GetId())

	return &concertpb.UpdateConcertResponse{
		Message: "Concert updated successfully",
	}, nil
}

func (s *ConcertService) DeleteConcert(ctx context.Context, req *concertpb.DeleteConcertRequest) (*concertpb.DeleteConcertResponse, error) {
	if req.GetId() == "" {
		return nil, fmt.Errorf("missing concert id")
	}

	if err := s.checkConcertPermission(ctx, req.GetId()); err != nil {
		return nil, err
	}

	if err := s.db.Where("concert_id = ?", req.GetId()).Delete(&models.ConcertItem{}).Error; err != nil {
		return nil, fmt.Errorf("failed to delete concert items: %v", err)
	}

	if err := s.db.Delete(&models.Concert{}, "id = ?", req.GetId()).Error; err != nil {
		return nil, fmt.Errorf("failed to delete concert: %v", err)
	}

	return &concertpb.DeleteConcertResponse{
		Message: "Concert deleted successfully",
	}, nil
}
