package models

import (
	"time"

	"gorm.io/gorm"
)

type Band struct {
	ID        uint           `gorm:"primaryKey"`
	ManagerID uint           `gorm:"not null;uniqueIndex:idx_band_name"`
	Name      string         `gorm:"not null;uniqueIndex:idx_band_name"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Manager User `gorm:"foreignKey:ManagerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type BandMember struct {
	ID        uint           `gorm:"primaryKey"`
	UserID    uint           `gorm:"not null;uniqueIndex:idx_user_band"`
	BandID    uint           `gorm:"not null;uniqueIndex:idx_user_band"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	User User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Band Band `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
