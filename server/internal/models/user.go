package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey"`
	PasswordHash string         `gorm:"not null"`
	Email        string         `gorm:"notnull;unique"`
	Name         string         `gorm:"not null"`
	Surname      string         `gorm:"not null"`
	Status       string         `gorm:"default:'pending'"`
	Role         string         `gorm:"default:'user'"`
	CreatedAt    time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
