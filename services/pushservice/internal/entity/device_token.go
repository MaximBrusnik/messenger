package entity

import (
	"time"

	"gorm.io/gorm"
)

// DeviceToken stores a push notification token per user/platform.
type DeviceToken struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Token     string         `gorm:"size:500;not null;uniqueIndex" json:"token"`
	Platform  string         `gorm:"size:20;not null" json:"platform"`
}
