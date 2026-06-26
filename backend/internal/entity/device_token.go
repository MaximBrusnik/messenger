package entity

import (
	"gorm.io/gorm"
	"time"
)

type DeviceToken struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID   uint   `gorm:"index;not null" json:"user_id"`
	Token    string `gorm:"size:500;not null" json:"token"`
	Platform string `gorm:"size:20;not null" json:"platform"` // android, ios
}

type RegisterDeviceRequest struct {
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=android ios"`
}
