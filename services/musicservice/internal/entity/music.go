package entity

import (
	"time"

	"gorm.io/gorm"
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

type Music struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Filename     string         `gorm:"size:255;not null" json:"-"`
	OriginalName string         `gorm:"size:255;not null" json:"original_name"`
	Title        string         `gorm:"size:255" json:"title"`
	Artist       string         `gorm:"size:255" json:"artist"`
	Size         int64          `json:"size"`
	MimeType     string         `gorm:"size:50" json:"mime_type"`
	UploadedBy   uint           `json:"uploaded_by"`
	Status       string         `gorm:"size:20;default:''" json:"status"`
}
