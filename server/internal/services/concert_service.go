package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/concertpb"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/auth"
	"github.com/ziomciopoziomcio/digital-music-stand/server/internal/models"
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
		breakStr := "0"
		if item.BreakMin != nil {
			breakStr = fmt.Sprintf("%d", *item.BreakMin)
		}
		h.Write([]byte(fmt.Sprintf(":%d-%s-%s", item.SortOrder, scoreStr, breakStr)))
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *ConcertService) CreateConcert(ctx context.Context, req *concertpb.CreateConcertRequest) (*concertpb.CreateConcertResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var bandID *uint
	if req.BandId != nil {
		bid := uint(req.GetBandId())
		bandID = &bid
	}

	concertID := req.GetId()
	if concertID == "" {
		concertID = uuid.New().String()
	}

	location := ""
	if req.Location != nil {
		location = *req.Location
	}
	startTime := ""
	if req.StartTime != nil {
		startTime = *req.StartTime
	}

	concert := models.Concert{
		ID:        concertID,
		UserID:    &userID,
		BandID:    bandID,
		Name:      req.GetName(),
		Location:  location,
		StartTime: startTime,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&concert).Error; err != nil {
			return err
		}
		for idx, item := range req.GetItems() {
			itemUUID := item.GetId()
			if itemUUID == "" {
				itemUUID = uuid.New().String()
			}
			var breakMin *int
			if item.BreakMin != nil {
				bm := int(*item.BreakMin)
				breakMin = &bm
			}

			concertItem := models.ConcertItem{
				ID:        itemUUID,
				ConcertID: concert.ID,
				SortOrder: idx + 1,
				ScoreID:   item.ScoreId,
				BreakMin:  breakMin,
			}
			if err := tx.Create(&concertItem).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create concert: %v", err)
	}

	checksum, _ := s.calculateConcertChecksum(concert.ID)

	return &concertpb.CreateConcertResponse{
		Id:       concert.ID,
		Message:  "Concert created successfully",
		Checksum: checksum,
	}, nil
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

	canEdit := false
	if !isOwner && !isBandManager {
		var suc models.SharedUserConcert
		if err := s.db.Where("concert_id = ? AND user_id = ? AND can_edit = ?", concert.ID, userID, true).First(&suc).Error; err == nil {
			canEdit = true
		} else {
			var sbc models.SharedBandConcert
			if err := s.db.Where("concert_id = ? AND can_edit = ? AND band_id IN (SELECT band_id FROM band_members WHERE user_id = ?)", concert.ID, true, userID).First(&sbc).Error; err == nil {
				canEdit = true
			}
		}
	}

	if !isOwner && !isBandManager && !canEdit {
		return nil, fmt.Errorf("permission denied: you cannot edit a concert owned by someone else")
	}

	concert.Name = req.GetName()
	if req.Location != nil {
		concert.Location = *req.Location
	}
	if req.StartTime != nil {
		concert.StartTime = *req.StartTime
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&concert).Error; err != nil {
			return err
		}

		if len(req.GetItems()) > 0 {
			if err := tx.Where("concert_id = ?", concert.ID).Delete(&models.ConcertItem{}).Error; err != nil {
				return err
			}

			for idx, item := range req.GetItems() {
				itemUUID := item.GetId()
				if itemUUID == "" {
					itemUUID = uuid.New().String()
				}
				var breakMin *int
				if item.BreakMin != nil {
					bm := int(*item.BreakMin)
					breakMin = &bm
				}

				concertItem := models.ConcertItem{
					ID:        itemUUID,
					ConcertID: concert.ID,
					SortOrder: idx + 1,
					ScoreID:   item.ScoreId,
					BreakMin:  breakMin,
				}
				if err := tx.Create(&concertItem).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update concert: %v", err)
	}

	checksum, _ := s.calculateConcertChecksum(concert.ID)

	return &concertpb.UpdateConcertResponse{
		Message:  "Concert updated successfully",
		Checksum: checksum,
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

func (s *ConcertService) ListMyConcerts(ctx context.Context, req *concertpb.ListMyConcertsRequest) (*concertpb.ListMyConcertsResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	subQueryDirect := s.db.Model(&models.SharedUserConcert{}).Select("concert_id").Where("user_id = ?", userID)
	subQueryBandShare := s.db.Model(&models.SharedBandConcert{}).Select("concert_id").Where("band_id IN (SELECT band_id FROM band_members WHERE user_id = ?)", userID)
	subQueryMyBands := s.db.Model(&models.BandMember{}).Select("band_id").Where("user_id = ?", userID)

	var concerts []models.Concert
	err = s.db.Where("user_id = ? OR band_id IN (?) OR id IN (?) OR id IN (?)", userID, subQueryMyBands, subQueryDirect, subQueryBandShare).
		Distinct().
		Find(&concerts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch concerts: %v", err)
	}

	var response []*concertpb.Concert
	for _, concert := range concerts {
		checksum, _ := s.calculateConcertChecksum(concert.ID)

		var items []models.ConcertItem
		s.db.Where("concert_id = ?", concert.ID).Order("sort_order asc").Find(&items)

		var pbItems []*concertpb.ConcertItem
		for _, item := range items {
			var scoreName *string
			var filePath *string
			if item.ScoreID != nil {
				var score models.Score
				if err := s.db.First(&score, "id = ?", *item.ScoreID).Error; err == nil {
					scoreName = &score.Name
					filePath = &score.FilePath
				}
			}

			pbItem := &concertpb.ConcertItem{
				Id:        item.ID,
				Order:     uint32(item.SortOrder),
				ScoreId:   item.ScoreID,
				ScoreName: scoreName,
				FilePath:  filePath,
			}
			if item.BreakMin != nil {
				bm := uint32(*item.BreakMin)
				pbItem.BreakMin = &bm
			}
			pbItems = append(pbItems, pbItem)
		}

		isOwner := false
		if concert.UserID != nil && *concert.UserID == userID {
			isOwner = true
		} else if concert.BandID != nil {
			var band models.Band
			if err := s.db.Where("id = ? AND manager_id = ?", *concert.BandID, userID).First(&band).Error; err == nil {
				isOwner = true
			}
		}

		canEdit := isOwner
		if !isOwner {
			var suc models.SharedUserConcert
			if err := s.db.Where("concert_id = ? AND user_id = ?", concert.ID, userID).First(&suc).Error; err == nil {
				if suc.CanEdit {
					canEdit = true
				}
			}
			if !canEdit {
				var sbc models.SharedBandConcert
				if err := s.db.Where("concert_id = ? AND can_edit = ? AND band_id IN (SELECT band_id FROM band_members WHERE user_id = ?)", concert.ID, true, userID).First(&sbc).Error; err == nil {
					canEdit = true
				}
			}
		}

		var bandID *uint32
		if concert.BandID != nil {
			bid := uint32(*concert.BandID)
			bandID = &bid
		}

		response = append(response, &concertpb.Concert{
			Id:        concert.ID,
			Name:      concert.Name,
			Location:  concert.Location,
			StartTime: concert.StartTime,
			Checksum:  checksum,
			BandId:    bandID,
			Items:     pbItems,
			IsOwner:   isOwner,
			CanEdit:   canEdit,
		})
	}

	return &concertpb.ListMyConcertsResponse{
		Concerts: response,
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
			CanEdit:   req.GetCanEdit(),
		}
		if err := s.db.Create(&sharedBand).Error; err != nil {
			return nil, fmt.Errorf("failed to share concert with band: %v", err)
		}

		return &concertpb.ShareConcertResponse{Message: "Concert shared with band successfully"}, nil
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
			CanEdit:      req.GetCanEdit(),
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
				CanEdit:   invite.CanEdit,
			}
			if err := tx.FirstOrCreate(&sharedConcert, sharedConcert).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to process concert invitation: %v", err)
	}

	return &concertpb.RespondToConcertInvitationResponse{Message: "Concert invitation processed successfully"}, nil
}
