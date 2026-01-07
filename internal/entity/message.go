package entity

import (
	"gorm.io/gorm"
	"time"
)

type Message struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	ChatID   uint   `gorm:"index;not null" json:"chat_id"`
	SenderID uint   `gorm:"index;not null" json:"sender_id"`
	Text     string `gorm:"type:text;not null" json:"text"`
	IsRead   bool   `gorm:"default:false" json:"is_read"`

	// Отношения
	Chat   Chat `gorm:"foreignKey:ChatID" json:"-"`
	Sender User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

type SendMessageRequest struct {
	Text string `json:"text" binding:"required,min=1,max=2000"`
}

type MessageResponse struct {
	ID        uint          `json:"id"`
	ChatID    uint          `json:"chat_id"`
	SenderID  uint          `json:"sender_id"`
	Text      string        `json:"text"`
	IsRead    bool          `json:"is_read"`
	CreatedAt time.Time     `json:"created_at"`
	Sender    *UserResponse `json:"sender,omitempty"`
}
