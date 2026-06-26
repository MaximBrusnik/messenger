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

	ChatID   uint       `gorm:"index;not null" json:"chat_id"`
	SenderID uint       `gorm:"index;not null" json:"sender_id"`
	Text     string     `gorm:"type:text;not null" json:"text"`
	IsRead   bool       `gorm:"default:false" json:"is_read"`
	ReadAt   *time.Time `json:"read_at,omitempty"`

	// Редактирование
	Edited   bool       `gorm:"default:false" json:"edited"`
	EditedAt *time.Time `json:"edited_at,omitempty"`

	// Системное сообщение (pin, unpin и т.д.)
	SystemType string `gorm:"size:20" json:"system_type,omitempty"`

	// Вложения
	AttachmentType string `gorm:"size:50" json:"attachment_type,omitempty"`
	AttachmentURL  string `gorm:"size:500" json:"attachment_url,omitempty"`
	AttachmentName string `gorm:"size:255" json:"attachment_name,omitempty"`
	AttachmentSize *int   `json:"attachment_size,omitempty"`

	// Отношения
	Chat      Chat              `gorm:"foreignKey:ChatID" json:"-"`
	Sender    User              `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Reactions []MessageReaction `gorm:"foreignKey:MessageID" json:"reactions,omitempty"`
}

type SendMessageRequest struct {
	Content        string `json:"content" binding:"max=2000"`
	AttachmentType string `json:"attachment_type,omitempty"`
	AttachmentURL  string `json:"attachment_url,omitempty"`
	AttachmentName string `json:"attachment_name,omitempty"`
	AttachmentSize *int   `json:"attachment_size,omitempty"`
}

type EditMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

type MessageResponse struct {
	ID             uint               `json:"id"`
	ChatID         uint               `json:"chat_id"`
	SenderID       uint               `json:"sender_id"`
	Text           string             `json:"text"`
	IsRead         bool               `json:"is_read"`
	ReadAt         *time.Time         `json:"read_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	Edited         bool               `json:"edited"`
	EditedAt       *time.Time         `json:"edited_at,omitempty"`
	Sender         *UserResponse      `json:"sender,omitempty"`
	Reactions      []ReactionResponse `json:"reactions,omitempty"`
	SystemType     string             `json:"system_type,omitempty"`
	AttachmentType string             `json:"attachment_type,omitempty"`
	AttachmentURL  string             `json:"attachment_url,omitempty"`
	AttachmentName string             `json:"attachment_name,omitempty"`
	AttachmentSize *int               `json:"attachment_size,omitempty"`
}
