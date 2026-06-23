package entity

import (
	"gorm.io/gorm"
	"time"
)

const MusicStatusPending = "pending"
const MusicStatusApproved = "approved"
const MusicStatusRejected = "rejected"

type AllowedMusicExt map[string]bool

var MusicExts = AllowedMusicExt{
	".mp3":  true,
	".wav":  true,
	".ogg":  true,
	".flac": true,
	".aac":  true,
	".wma":  true,
}

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

type MusicResponse struct {
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Artist       string    `json:"artist"`
	OriginalName string    `json:"original_name"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	UploadedBy   uint      `json:"uploaded_by"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type MusicUploadResponse struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}
