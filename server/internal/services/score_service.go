package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
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
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id from context: %v", err)
	}

	if req.GetName() == "" || len(req.GetFileData()) == 0 {
		return nil, fmt.Errorf("missing required fields")
	}

	fileID := uuid.New().String()
	ext := req.GetFileExtension()
	if ext == "" {
		ext = ".pdf"
	}

	storageDir := "./uploads/scores"
	if err := os.MkdirAll(storageDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	filePath := filepath.Join(storageDir, fmt.Sprintf("%s%s", fileID, ext))

	if err := os.WriteFile(filePath, req.GetFileData(), 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	hash := sha256.Sum256(req.GetFileData())
	checksum := hex.EncodeToString(hash[:])

	var composer *string
	if req.GetComposer() != "" {
		composer = &req.Composer
	}

	score := models.Score{
		Name:          req.GetName(),
		Composer:      composer,
		FilePath:      filePath,
		Checksum:      checksum,
		OwnerID:       uint(userID),
		FileExtension: ext,
	}

	if err := s.db.Create(&score).Error; err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save score in database: %v", err)
	}

	return &scorepb.CreateScoreResponse{
		Id:      uint32(score.ID),
		Message: "Score created successfully",
	}, nil
}

func (s *ScoreService) ListMyScores(ctx context.Context, req *scorepb.ListMyScoresRequest) (*scorepb.ListMyScoresResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id from context: %v", err)
	}

	var scores []models.Score
	if err := s.db.Where("owner_id = ?", userID).Find(&scores).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch scores: %v", err)
	}

	var scoreList []*scorepb.Score
	for _, score := range scores {
		var composer string

		if score.Composer == nil {
			composer = "-"
		} else {
			composer = *score.Composer
		}

		scoreList = append(scoreList, &scorepb.Score{
			Id:            uint32(score.ID),
			Name:          score.Name,
			Composer:      composer,
			FilePath:      score.FilePath,
			Checksum:      score.Checksum,
			FileExtension: score.FileExtension,
		})
	}

	return &scorepb.ListMyScoresResponse{
		Scores: scoreList,
	}, nil
}

func (s *ScoreService) DownloadScore(req *scorepb.DownloadScoreRequest, stream scorepb.ScoreService_DownloadScoreServer) error {
	var score models.Score
	if err := s.db.First(&score, req.GetScoreId()).Error; err != nil {
		return fmt.Errorf("score not found: %v", err)
	}

	file, err := os.Open(score.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	buffer := make([]byte, 64*1024)
	for {
		bytesRead, err := file.Read(buffer)
		if bytesRead > 0 {
			if sendErr := stream.Send(&scorepb.DownloadScoreResponse{
				ChunkData: buffer[:bytesRead],
			}); sendErr != nil {
				return fmt.Errorf("failed to send chunk: %v", sendErr)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
	}

	return nil
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
