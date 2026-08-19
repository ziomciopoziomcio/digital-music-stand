package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Score struct {
	ID            string `gorm:"primaryKey;type:varchar(36)"`
	OwnerID       uint   `gorm:"not null"`
	Name          string `gorm:"not null"`
	Composer      *string
	FilePath      string `gorm:"not null"`
	FileExtension string `gorm:"not null;default:'.pdf'"`
	Checksum      string `gorm:"not null"`

	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Owner           User              `gorm:"foreignKey:OwnerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	SharedWithUsers []SharedUserScore `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	SharedWithBands []SharedBandScore `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (s *Score) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

type SharedUserScore struct {
	ID        uint      `gorm:"primaryKey"`
	ScoreID   string    `gorm:"not null;type:varchar(36);uniqueIndex:idx_shared_user_score"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_shared_user_score"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	Score Score `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User  User  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type SharedBandScore struct {
	ID        uint      `gorm:"primaryKey"`
	ScoreID   string    `gorm:"not null;type:varchar(36);uniqueIndex:idx_shared_band_score"`
	BandID    uint      `gorm:"not null;uniqueIndex:idx_shared_band_score"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	Score Score `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Band  Band  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type ShareScoreInvitation struct {
	ID           uint      `gorm:"primaryKey"`
	ScoreID      string    `gorm:"not null;type:varchar(36);uniqueIndex:idx_score_invite"`
	InviteeEmail string    `gorm:"not null;uniqueIndex:idx_score_invite"`
	Status       string    `gorm:"not null;default:'pending'"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	Score Score `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
