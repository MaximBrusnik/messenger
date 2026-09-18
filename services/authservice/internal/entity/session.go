package entity

import "time"

type Session struct {
	ID                string    `gorm:"size:64;primarykey" json:"id"`
	UserID            uint      `gorm:"index;not null" json:"user_id"`
	DeviceName        string    `gorm:"size:100" json:"device_name"`
	Platform          string    `gorm:"size:50" json:"platform"`
	DeviceFingerprint string    `gorm:"size:200;index" json:"device_fingerprint"`
	IP                string    `gorm:"size:64" json:"ip"`
	TokenExpiresAt    time.Time `json:"token_expires_at"`
	LastLoginAt       time.Time `json:"last_login_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
