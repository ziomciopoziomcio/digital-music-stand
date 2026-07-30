package models

import (
	"time"

	"gorm.io/gorm"
)

type Score struct {
	ID        uint   `gorm:"primaryKey"`
	FilePath  string `gorm:"not null"`
	OwnerID   uint   `gorm:"not null;uniqueIndex:idx_owner_name"`
	Name      string `gorm:"not null;uniqueIndex:idx_owner_name"`
	Composer  *string
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type SharedUserScore struct{}
