package services

import (
	"context"
	"fmt"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
	"gorm.io/gorm"
)

type ScoreService struct {
	scorepb.UnimplementedScoreServiceServer
	db *gorm.DB
}

func NewScoreService(db *gorm.DB) *ScoreService {
	return &ScoreService{
		db: db,
	}
}

func (s *ScoreService) CreateScore(ctx context.Context, req *scorepb.CreateScoreRequest) (*scorepb.CreateScoreResponse, error) {
	if req.GetName() == "" || req.GetFilePath() == "" || req.GetOwnerId() == 0 { // todo: OwnerID should be extracted from JWT token
		return nil, fmt.Errorf("missing required fields")
	}

	var composer *string
	if req.GetComposer() != "" {
		composer = &req.Composer
	}

	newScore := models.Score{
		Name:     req.GetName(),
		Composer: composer,
		FilePath: req.GetFilePath(),
		OwnerID:  uint(req.GetOwnerId()),
	}

	if err := s.db.Create(&newScore).Error; err != nil {
		return nil, fmt.Errorf("failed to create score: %v", err)
	}

	return &scorepb.CreateScoreResponse{
		Id:      uint32(newScore.ID),
		Message: "Score created successfully",
	}, nil
}

func (s *ScoreService) ListMyScores(ctx context.Context, req *scorepb.ListMyScoresRequest) (*scorepb.ListMyScoresResponse, error) {
	if req.GetUserId() == 0 { // todo: UserID should be extracted from JWT token
		return nil, fmt.Errorf("missing required fields")
	}

	var scores []models.Score
	if err := s.db.Where("owner_id = ?", req.GetUserId()).Find(&scores).Error; err != nil {
		return nil, fmt.Errorf("failed to list scores: %v", err)
	}

	var scoreList []*scorepb.Score
	for _, score := range scores {
		var composer string
		if score.Composer != nil {
			composer = *score.Composer
		}
		scoreList = append(scoreList, &scorepb.Score{
			Id:       uint32(score.ID),
			Name:     score.Name,
			Composer: composer,
			FilePath: score.FilePath,
		})
	}

	return &scorepb.ListMyScoresResponse{
		Scores: scoreList,
	}, nil
}

func (s *ScoreService) ShareScore(ctx context.Context, req *scorepb.ShareScoreRequest) (*scorepb.ShareScoreResponse, error) {
	if req.GetScoreId() == 0 {
		return nil, fmt.Errorf("missing required fields")
	}
	if req.UserId == nil && req.BandId == nil {
		return nil, fmt.Errorf("either user_id or band_id must be provided")
	}
	if req.UserId != nil && req.BandId != nil {
		return nil, fmt.Errorf("only one of user_id or band_id can be provided")
	}

	var score models.Score
	if err := s.db.First(&score, req.GetScoreId()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("score not found")
		}
		return nil, fmt.Errorf("failed to query score: %v", err)
	}

	if req.UserId != nil {
		newShare := models.SharedUserScore{
			ScoreID: uint(req.GetScoreId()),
			UserID:  uint(*req.UserId),
		}
		if err := s.db.Create(&newShare).Error; err != nil {
			return nil, fmt.Errorf("failed to share score with user: %v", err)
		}
	} else if req.BandId != nil {
		newShare := models.SharedBandScore{
			ScoreID: uint(req.GetScoreId()),
			BandID:  uint(*req.BandId),
		}
		if err := s.db.Create(&newShare).Error; err != nil {
			return nil, fmt.Errorf("failed to share score with band: %v", err)
		}
	}

	return &scorepb.ShareScoreResponse{
		Message: "Score shared successfully",
	}, nil
}
