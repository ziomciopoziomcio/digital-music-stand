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
	FileExtension string `gorm:"not null"`
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
	ID        uint   `gorm:"primaryKey"`
	ScoreID   string `gorm:"not null;type:varchar(36);index:idx_user_score,unique"`
	UserID    uint   `gorm:"not null;index:idx_user_score,unique"`
	CreatedAt time.Time

	Score Score `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User  User  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type SharedBandScore struct {
	ID        uint   `gorm:"primaryKey"`
	ScoreID   string `gorm:"not null;type:varchar(36);index:idx_band_score,unique"`
	BandID    uint   `gorm:"not null;index:idx_band_score,unique"`
	CreatedAt time.Time

	Score Score `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Band  Band  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
