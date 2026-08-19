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
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var concert models.Concert
	if err := s.db.Where("id = ?", req.GetId()).First(&concert).Error; err != nil {
		return nil, fmt.Errorf("concert not found")
	}

	isOwner := concert.UserID != nil && *concert.UserID == userID
	isBandManager := false
	if concert.BandID != nil {
		var band models.Band
		if err := s.db.Where("id = ? AND manager_id = ?", *concert.BandID, userID).First(&band).Error; err == nil {
			isBandManager = true
		}
	}

	if !isOwner && !isBandManager {
		return nil, fmt.Errorf("permission denied: you cannot edit a concert owned by someone else")
	}

	concert.Name = req.GetName()
	if req.Location != nil {
		concert.Location = *req.Location
	}
	if req.StartTime != nil {
		concert.StartTime = *req.StartTime
	}

	if err := s.db.Save(&concert).Error; err != nil {
		return nil, fmt.Errorf("failed to update concert: %v", err)
	}

	return &concertpb.UpdateConcertResponse{
		Message:  "Concert updated successfully",
		Checksum: concert.Checksum,
	}, nil
}

func (s *ConcertService) DeleteConcert(ctx context.Context, req *concertpb.DeleteConcertRequest) (*concertpb.DeleteConcertResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var concert models.Concert
	if err := s.db.Where("id = ?", req.GetId()).First(&concert).Error; err != nil {
		return nil, fmt.Errorf("concert not found")
	}

	isOwner := concert.UserID != nil && *concert.UserID == userID
	isBandManager := false
	if concert.BandID != nil {
		var band models.Band
		if err := s.db.Where("id = ? AND manager_id = ?", *concert.BandID, userID).First(&band).Error; err == nil {
			isBandManager = true
		}
	}

	if !isOwner && !isBandManager {
		return nil, fmt.Errorf("permission denied: you cannot delete a concert owned by someone else")
	}

	if err := s.db.Delete(&concert).Error; err != nil {
		return nil, fmt.Errorf("failed to delete concert: %v", err)
	}

	return &concertpb.DeleteConcertResponse{
		Message: "Concert deleted successfully",
	}, nil
}

func (s *ConcertService) ShareConcert(ctx context.Context, req *concertpb.ShareConcertRequest) (*concertpb.ShareConcertResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var concert models.Concert
	if err := s.db.Where("id = ? AND (user_id = ? OR band_id IN (SELECT band_id FROM band_members WHERE user_id = ?))", req.GetConcertId(), userID, userID).First(&concert).Error; err != nil {
		return nil, fmt.Errorf("concert not found or permission denied")
	}

	if req.TargetBandId != nil {
		var member models.BandMember
		if err := s.db.Where("band_id = ? AND user_id = ?", *req.TargetBandId, userID).First(&member).Error; err != nil {
			return nil, fmt.Errorf("you are not a member of this band")
		}

		var existingShare models.SharedBandConcert
		if err := s.db.Where("concert_id = ? AND band_id = ?", concert.ID, *req.TargetBandId).First(&existingShare).Error; err == nil {
			return nil, fmt.Errorf("concert is already shared with this band")
		}

		sharedBand := models.SharedBandConcert{
			ConcertID: concert.ID,
			BandID:    uint(*req.TargetBandId),
		}
		if err := s.db.Create(&sharedBand).Error; err != nil {
			return nil, fmt.Errorf("failed to share concert with band: %v", err)
		}

		var items []models.ConcertItem
		s.db.Where("concert_id = ? AND score_id IS NOT NULL", concert.ID).Find(&items)
		for _, item := range items {
			if item.ScoreID != nil {
				sharedBandScore := models.SharedBandScore{
					ScoreID: *item.ScoreID,
					BandID:  uint(*req.TargetBandId),
				}
				s.db.FirstOrCreate(&sharedBandScore, sharedBandScore)
			}
		}

		return &concertpb.ShareConcertResponse{Message: "Concert and its scores shared with band successfully"}, nil
	}

	if req.TargetEmail != nil {
		email := *req.TargetEmail

		var targetUser models.User
		if err := s.db.Where("email = ?", email).First(&targetUser).Error; err != nil {
			return nil, fmt.Errorf("user with email %s does not exist", email)
		}

		if targetUser.ID == userID {
			return nil, fmt.Errorf("you cannot share a concert with yourself")
		}

		var existingShare models.SharedUserConcert
		if err := s.db.Where("concert_id = ? AND user_id = ?", concert.ID, targetUser.ID).First(&existingShare).Error; err == nil {
			return nil, fmt.Errorf("this user already has access to this concert")
		}

		var existingInvite models.ShareConcertInvitation
		if err := s.db.Where("concert_id = ? AND invitee_email = ? AND status = ?", concert.ID, email, "pending").First(&existingInvite).Error; err == nil {
			return nil, fmt.Errorf("an invitation is already pending for this user")
		}

		invite := models.ShareConcertInvitation{
			ConcertID:    concert.ID,
			InviteeEmail: email,
			Status:       "pending",
		}
		if err := s.db.Create(&invite).Error; err != nil {
			return nil, fmt.Errorf("failed to create concert invitation: %v", err)
		}
		return &concertpb.ShareConcertResponse{Message: "Concert invitation sent to user"}, nil
	}

	return nil, fmt.Errorf("target_email or target_band_id must be provided")
}

func (s *ConcertService) RevokeConcertAccess(ctx context.Context, req *concertpb.RevokeConcertAccessRequest) (*concertpb.RevokeConcertAccessResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var concert models.Concert
	if err := s.db.Where("id = ? AND user_id = ?", req.GetConcertId(), userID).First(&concert).Error; err != nil {
		return nil, fmt.Errorf("concert not found or permission denied")
	}

	if req.TargetBandId != nil {
		res := s.db.Where("concert_id = ? AND band_id = ?", concert.ID, *req.TargetBandId).Delete(&models.SharedBandConcert{})
		if res.RowsAffected == 0 {
			return nil, fmt.Errorf("this band does not have access to this concert")
		}
		return &concertpb.RevokeConcertAccessResponse{Message: "Concert access revoked for band"}, nil
	}

	if req.TargetEmail != nil {
		email := *req.TargetEmail
		var deletedShares int64
		var deletedInvites int64

		var targetUser models.User
		if err := s.db.Where("email = ?", email).First(&targetUser).Error; err == nil {
			res := s.db.Where("concert_id = ? AND user_id = ?", concert.ID, targetUser.ID).Delete(&models.SharedUserConcert{})
			deletedShares = res.RowsAffected
		}

		res := s.db.Where("concert_id = ? AND invitee_email = ? AND status = 'pending'", concert.ID, email).Delete(&models.ShareConcertInvitation{})
		deletedInvites = res.RowsAffected

		if deletedShares == 0 && deletedInvites == 0 {
			return nil, fmt.Errorf("no active access or pending invitation found for %s", email)
		}

		return &concertpb.RevokeConcertAccessResponse{Message: "Concert access revoked successfully"}, nil
	}

	return nil, fmt.Errorf("target_email or target_band_id must be provided")
}

func (s *ConcertService) ListConcertInvitations(ctx context.Context, req *concertpb.ListConcertInvitationsRequest) (*concertpb.ListConcertInvitationsResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	var invites []models.ShareConcertInvitation
	if err := s.db.Preload("Concert.User").Where("invitee_email = ? AND status = ?", user.Email, "pending").Find(&invites).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch concert invitations: %v", err)
	}

	var response []*concertpb.ConcertInvitation
	for _, inv := range invites {
		ownerEmail := "Unknown"
		if inv.Concert.User.Email != "" {
			ownerEmail = inv.Concert.User.Email
		}
		response = append(response, &concertpb.ConcertInvitation{
			Id:          uint32(inv.ID),
			ConcertId:   inv.ConcertID,
			ConcertName: inv.Concert.Name,
			OwnerEmail:  ownerEmail,
		})
	}

	return &concertpb.ListConcertInvitationsResponse{Invitations: response}, nil
}

func (s *ConcertService) RespondToConcertInvitation(ctx context.Context, req *concertpb.RespondToConcertInvitationRequest) (*concertpb.RespondToConcertInvitationResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	var invite models.ShareConcertInvitation
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
			sharedConcert := models.SharedUserConcert{
				ConcertID: invite.ConcertID,
				UserID:    userID,
			}
			if err := tx.FirstOrCreate(&sharedConcert, sharedConcert).Error; err != nil {
				return err
			}

			// Automatycznie przyznajemy również dostęp do wszystkich utworów wchodzących w skład koncertu
			var items []models.ConcertItem
			if err := tx.Where("concert_id = ? AND score_id IS NOT NULL", invite.ConcertID).Find(&items).Error; err == nil {
				for _, item := range items {
					if item.ScoreID != nil {
						sharedScore := models.SharedUserScore{
							ScoreID: *item.ScoreID,
							UserID:  userID,
						}
						_ = tx.FirstOrCreate(&sharedScore, sharedScore)
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to process concert invitation: %v", err)
	}

	return &concertpb.RespondToConcertInvitationResponse{Message: "Concert invitation processed successfully"}, nil
}
