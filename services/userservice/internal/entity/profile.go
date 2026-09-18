package entity

import (
	"time"

	"gorm.io/gorm"
)

type Profile struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Username         string         `gorm:"uniqueIndex;size:100;not null" json:"username"`
	Email            string         `gorm:"uniqueIndex;size:255;not null" json:"email"`
	EmailVerified    bool           `gorm:"default:false" json:"email_verified"`
	IsBot            bool           `gorm:"default:false" json:"is_bot"`
	IsAdmin          bool           `gorm:"default:false" json:"is_admin"`
	Avatar           string         `gorm:"size:500" json:"avatar,omitempty"`
	Status           string         `gorm:"size:50;default:'offline'" json:"status"`
	Bio              string         `gorm:"size:500" json:"bio,omitempty"`
	DateOfBirth      *time.Time     `json:"date_of_birth,omitempty"`
	ShowOnlineStatus bool           `gorm:"default:true" json:"show_online_status"`
	LastSeenPrivacy  string         `gorm:"size:20;default:'everyone'" json:"last_seen_privacy"`
	AvatarPrivacy    string         `gorm:"size:20;default:'everyone'" json:"avatar_privacy"`
	SoundEnabled     bool           `gorm:"default:true" json:"sound_enabled"`
	LastLogin        time.Time      `json:"last_login,omitempty"`
}

type Contact struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	ContactID uint      `gorm:"primaryKey" json:"contact_id"`
	CreatedAt time.Time `json:"created_at"`
}
