package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Concert struct {
	ID        string         `gorm:"primaryKey;type:varchar(36)"`
	BandID    *uint          `gorm:"index"`
	UserID    *uint          `gorm:"index"`
	Name      string         `gorm:"not null"`
	Location  string         `gorm:"default:''"`
	StartTime string         `gorm:"default:''"`
	Checksum  string         `gorm:"not null;default:''"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Band Band `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (c *Concert) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

type ConcertItem struct {
	ID        string  `gorm:"primaryKey;type:varchar(36)"`
	ConcertID string  `gorm:"not null;type:varchar(36);index"`
	SortOrder int     `gorm:"not null"`
	ScoreID   *string `gorm:"type:varchar(36)"`
	BreakMin  *int
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Concert Concert `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Score   Score   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (ci *ConcertItem) BeforeCreate(tx *gorm.DB) error {
	if ci.ID == "" {
		ci.ID = uuid.New().String()
	}
	return nil
}
