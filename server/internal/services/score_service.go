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

	scoreID := uuid.New().String()
	if req.Id != nil && *req.Id != "" {
		scoreID = *req.Id
	}

	ext := req.GetFileExtension()
	if ext == "" {
		ext = ".pdf"
	}

	storageDir := "./uploads/scores"
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	filePath := filepath.Join(storageDir, scoreID+ext)
	if err := os.WriteFile(filePath, req.GetFileData(), 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	hash := sha256.Sum256(req.GetFileData())
	checksum := hex.EncodeToString(hash[:])

	var composer *string
	if req.GetComposer() != "" {
		comp := req.GetComposer()
		composer = &comp
	}

	newScore := models.Score{
		ID:            scoreID,
		OwnerID:       userID,
		Name:          req.GetName(),
		Composer:      composer,
		FilePath:      filePath,
		FileExtension: ext,
		Checksum:      checksum,
	}

	if err := s.db.Create(&newScore).Error; err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save score to database: %v", err)
	}

	return &scorepb.CreateScoreResponse{
		Id:       newScore.ID,
		Message:  "Score uploaded successfully",
		Checksum: checksum,
	}, nil
}

func (s *ScoreService) ListMyScores(ctx context.Context, req *scorepb.ListMyScoresRequest) (*scorepb.ListMyScoresResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user id: %v", err)
	}

	var scores []models.Score
	subQueryDirect := s.db.Model(&models.SharedUserScore{}).Select("score_id").Where("user_id = ?", userID)
	subQueryBand := s.db.Model(&models.SharedBandScore{}).Select("score_id").Where("band_id IN (?)", s.db.Model(&models.BandMember{}).Select("band_id").Where("user_id = ?", userID))

	err = s.db.Where("owner_id = ? OR id IN (?) OR id IN (?)", userID, subQueryDirect, subQueryBand).
		Distinct().
		Find(&scores).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch scores: %v", err)
	}

	var scoreList []*scorepb.Score
	for _, score := range scores {
		composer := ""
		if score.Composer != nil {
			composer = *score.Composer
		}

		scoreList = append(scoreList, &scorepb.Score{
			Id:            score.ID,
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
	if req.GetScoreId() == "" {
		return fmt.Errorf("missing score_id")
	}

	var score models.Score
	if err := s.db.First(&score, "id = ?", req.GetScoreId()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("score not found")
		}
		return fmt.Errorf("failed to query score: %v", err)
	}

	file, err := os.Open(score.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	buffer := make([]byte, 64*1024)
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			sendErr := stream.Send(&scorepb.DownloadScoreResponse{
				ChunkData: buffer[:n],
			})
			if sendErr != nil {
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

func (s *ScoreService) UpdateScore(ctx context.Context, req *scorepb.UpdateScoreRequest) (*scorepb.UpdateScoreResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var score models.Score
	if err := s.db.Where("id = ?", req.GetId()).First(&score).Error; err != nil {
		return nil, fmt.Errorf("score not found")
	}

	if score.OwnerID != userID {
		return nil, fmt.Errorf("permission denied: you cannot edit a score owned by someone else")
	}

	if req.GetName() != "" {
		score.Name = req.GetName()
	}
	if req.Composer != nil {
		score.Composer = req.Composer
	}

	if err := s.db.Save(&score).Error; err != nil {
		return nil, fmt.Errorf("failed to update score: %v", err)
	}

	return &scorepb.UpdateScoreResponse{
		Message:  "Score updated successfully",
		Checksum: score.Checksum,
	}, nil
}

func (s *ScoreService) DeleteScore(ctx context.Context, req *scorepb.DeleteScoreRequest) (*scorepb.DeleteScoreResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var score models.Score
	if err := s.db.Where("id = ?", req.GetId()).First(&score).Error; err != nil {
		return nil, fmt.Errorf("score not found")
	}

	if score.OwnerID != userID {
		return nil, fmt.Errorf("permission denied: you cannot delete a score owned by someone else")
	}

	if err := s.db.Delete(&score).Error; err != nil {
		return nil, fmt.Errorf("failed to delete score: %v", err)
	}

	return &scorepb.DeleteScoreResponse{
		Message: "Score deleted successfully",
	}, nil
}

func (s *ScoreService) ShareScore(ctx context.Context, req *scorepb.ShareScoreRequest) (*scorepb.ShareScoreResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var score models.Score
	if err := s.db.Where("id = ? AND owner_id = ?", req.GetScoreId(), userID).First(&score).Error; err != nil {
		return nil, fmt.Errorf("score not found or permission denied")
	}

	if req.TargetBandId != nil {
		var member models.BandMember
		if err := s.db.Where("band_id = ? AND user_id = ?", *req.TargetBandId, userID).First(&member).Error; err != nil {
			return nil, fmt.Errorf("you are not a member of this band")
		}

		var existingShare models.SharedBandScore
		if err := s.db.Where("score_id = ? AND band_id = ?", score.ID, *req.TargetBandId).First(&existingShare).Error; err == nil {
			return nil, fmt.Errorf("score is already shared with this band")
		}

		sharedBand := models.SharedBandScore{
			ScoreID: score.ID,
			BandID:  uint(*req.TargetBandId),
		}
		if err := s.db.Create(&sharedBand).Error; err != nil {
			return nil, fmt.Errorf("failed to share score with band: %v", err)
		}
		return &scorepb.ShareScoreResponse{Message: "Score shared with band successfully"}, nil
	}

	if req.TargetEmail != nil {
		email := *req.TargetEmail

		var targetUser models.User
		if err := s.db.Where("email = ?", email).First(&targetUser).Error; err != nil {
			return nil, fmt.Errorf("user with email %s does not exist", email)
		}

		if targetUser.ID == userID {
			return nil, fmt.Errorf("you cannot share a score with yourself")
		}

		var existingShare models.SharedUserScore
		if err := s.db.Where("score_id = ? AND user_id = ?", score.ID, targetUser.ID).First(&existingShare).Error; err == nil {
			return nil, fmt.Errorf("this user already has access to this score")
		}

		var existingInvite models.ShareScoreInvitation
		if err := s.db.Where("score_id = ? AND invitee_email = ? AND status = ?", score.ID, email, "pending").First(&existingInvite).Error; err == nil {
			return nil, fmt.Errorf("an invitation is already pending for this user")
		}

		invite := models.ShareScoreInvitation{
			ScoreID:      score.ID,
			InviteeEmail: email,
			Status:       "pending",
		}
		if err := s.db.Create(&invite).Error; err != nil {
			return nil, fmt.Errorf("failed to create score invitation: %v", err)
		}
		return &scorepb.ShareScoreResponse{Message: "Score sharing invitation sent to user"}, nil
	}

	return nil, fmt.Errorf("target_email or target_band_id must be provided")
}

func (s *ScoreService) RevokeScoreAccess(ctx context.Context, req *scorepb.RevokeScoreAccessRequest) (*scorepb.RevokeScoreAccessResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var score models.Score
	if err := s.db.Where("id = ? AND owner_id = ?", req.GetScoreId(), userID).First(&score).Error; err != nil {
		return nil, fmt.Errorf("score not found or permission denied")
	}

	if req.TargetBandId != nil {
		res := s.db.Where("score_id = ? AND band_id = ?", score.ID, *req.TargetBandId).Delete(&models.SharedBandScore{})
		if res.RowsAffected == 0 {
			return nil, fmt.Errorf("this band does not have access to this score")
		}
		return &scorepb.RevokeScoreAccessResponse{Message: "Access revoked for band"}, nil
	}

	if req.TargetEmail != nil {
		email := *req.TargetEmail
		var deletedShares int64
		var deletedInvites int64

		var targetUser models.User
		if err := s.db.Where("email = ?", email).First(&targetUser).Error; err == nil {
			res := s.db.Where("score_id = ? AND user_id = ?", score.ID, targetUser.ID).Delete(&models.SharedUserScore{})
			deletedShares = res.RowsAffected
		}

		res := s.db.Where("score_id = ? AND invitee_email = ? AND status = 'pending'", score.ID, email).Delete(&models.ShareScoreInvitation{})
		deletedInvites = res.RowsAffected

		if deletedShares == 0 && deletedInvites == 0 {
			return nil, fmt.Errorf("no active access or pending invitation found for %s", email)
		}

		return &scorepb.RevokeScoreAccessResponse{Message: "Access revoked successfully"}, nil
	}

	return nil, fmt.Errorf("target_email or target_band_id must be provided")
}

func (s *ScoreService) ListScoreInvitations(ctx context.Context, req *scorepb.ListScoreInvitationsRequest) (*scorepb.ListScoreInvitationsResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	var invites []models.ShareScoreInvitation
	if err := s.db.Preload("Score.Owner").Where("invitee_email = ? AND status = ?", user.Email, "pending").Find(&invites).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch score invitations: %v", err)
	}

	var response []*scorepb.ScoreInvitation
	for _, inv := range invites {
		response = append(response, &scorepb.ScoreInvitation{
			Id:         uint32(inv.ID),
			ScoreId:    fmt.Sprintf("%s", inv.ScoreID),
			ScoreName:  inv.Score.Name,
			OwnerEmail: inv.Score.Owner.Email,
		})
	}

	return &scorepb.ListScoreInvitationsResponse{Invitations: response}, nil
}

func (s *ScoreService) RespondToScoreInvitation(ctx context.Context, req *scorepb.RespondToScoreInvitationRequest) (*scorepb.RespondToScoreInvitationResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	var invite models.ShareScoreInvitation
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
			sharedScore := models.SharedUserScore{
				ScoreID: invite.ScoreID,
				UserID:  userID,
			}
			if err := tx.FirstOrCreate(&sharedScore, sharedScore).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to process score invitation: %v", err)
	}

	return &scorepb.RespondToScoreInvitationResponse{Message: "Score invitation processed successfully"}, nil
}
