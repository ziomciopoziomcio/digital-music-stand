package models

import (
	"time"

	"gorm.io/gorm"
)

type Concert struct {
	ID        uint           `gorm:"primaryKey"`
	BandID    *uint          `gorm:"uniqueIndex:idx_band_name"`
	UserID    *uint          `gorm:"uniqueIndex:idx_band_name"`
	Name      string         `gorm:"not null;uniqueIndex:idx_band_name;uniqueIndex:idx_user_name"`
	Location  string         `gorm:"default:''"`
	StartTime string         `gorm:"default:''"`
	Checksum  string         `gorm:"not null;default:''"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Band Band `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type ConcertItem struct {
	ID        uint `gorm:"primaryKey"`
	ConcertID uint `gorm:"not null;uniqueIndex:idx_concert_sort"`
	SortOrder int  `gorm:"not null;uniqueIndex:idx_concert_sort"`
	ScoreID   *uint
	BreakMin  *int
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Concert Concert `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Score   Score   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
