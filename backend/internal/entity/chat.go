package entity

import (
	"gorm.io/gorm"
	"time"
)

type Chat struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name   string `gorm:"size:255" json:"name,omitempty"`
	Type   string `gorm:"size:20;default:'private'" json:"type"` // private, group
	Avatar string `gorm:"size:500" json:"avatar,omitempty"`

	// Отношения
	Participants []User    `gorm:"many2many:chat_users;" json:"participants,omitempty"`
	Messages     []Message `gorm:"foreignKey:ChatID" json:"messages,omitempty"`
	LastMessage  *Message  `gorm:"-" json:"last_message,omitempty"`
	UnreadCount  int       `gorm:"-" json:"unread,omitempty"`

	//Закрепления сообщений
	PinnedMessageID *uint    `gorm:"default:null" json:"pinned_message_id,omitempty"`
	PinnedMessage   *Message `gorm:"-" json:"pinned_message,omitempty"`
}

type ChatUser struct {
	ChatID   uint      `gorm:"primaryKey" json:"chat_id"`
	UserID   uint      `gorm:"primaryKey" json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
	IsAdmin  bool      `gorm:"default:false" json:"is_admin"`
	LastRead time.Time `json:"last_read,omitempty"`
}

// DTO для запросов
type CreateChatRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"`
}

type ChatResponse struct {
	ID            uint             `json:"id"`
	Name          string           `json:"name,omitempty"`
	Type          string           `json:"type"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Participants  []UserResponse   `json:"participants,omitempty"`
	LastMessage   *MessageResponse `json:"last_message,omitempty"`
	Unread        int              `json:"unread,omitempty"`
	PinnedMessage *MessageResponse `json:"pinned_message,omitempty"`
}
