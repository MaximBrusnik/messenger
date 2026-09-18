package entity

import (
	"time"

	"gorm.io/gorm"
)

type Chat struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Name            string         `gorm:"size:255" json:"name,omitempty"`
	Type            string         `gorm:"size:20;default:'private'" json:"type"`
	Avatar          string         `gorm:"size:500" json:"avatar,omitempty"`
	PinnedMessageID *uint          `gorm:"default:null" json:"pinned_message_id,omitempty"`
}

type ChatUser struct {
	ChatID     uint      `gorm:"primaryKey" json:"chat_id"`
	UserID     uint      `gorm:"primaryKey" json:"user_id"`
	JoinedAt   time.Time `json:"joined_at"`
	IsAdmin    bool      `gorm:"default:false" json:"is_admin"`
	IsArchived bool      `gorm:"default:false" json:"is_archived"`
	LastRead   time.Time `json:"last_read,omitempty"`
}

type Message struct {
	ID                     uint           `gorm:"primarykey" json:"id"`
	ChatID                 uint           `gorm:"index:idx_msg_chat_created,priority:1;not null" json:"chat_id"`
	CreatedAt              time.Time      `gorm:"index:idx_msg_chat_created,priority:2" json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
	SenderID               uint           `gorm:"index;not null" json:"sender_id"`
	Text                   string         `gorm:"type:text;not null" json:"text"`
	IsRead                 bool           `gorm:"default:false" json:"is_read"`
	ReadAt                 *time.Time     `json:"read_at,omitempty"`
	Edited                 bool           `gorm:"default:false" json:"edited"`
	EditedAt               *time.Time     `json:"edited_at,omitempty"`
	SystemType             string         `gorm:"size:20" json:"system_type,omitempty"`
	AttachmentType         string         `gorm:"size:50" json:"attachment_type,omitempty"`
	AttachmentURL          string         `gorm:"size:500" json:"attachment_url,omitempty"`
	AttachmentName         string         `gorm:"size:255" json:"attachment_name,omitempty"`
	AttachmentSize         *int           `json:"attachment_size,omitempty"`
	IsForwarded            bool           `gorm:"default:false" json:"is_forwarded"`
	ForwardedFromSenderID  uint           `json:"forwarded_from_sender_id,omitempty"`
	ForwardedFromChatID    uint           `json:"forwarded_from_chat_id,omitempty"`
	ForwardedFromMessageID uint           `json:"forwarded_from_message_id,omitempty"`
}

type MessageReaction struct {
	MessageID uint      `gorm:"primaryKey" json:"message_id"`
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	Reaction  string    `gorm:"primaryKey;size:50" json:"reaction"`
	CreatedAt time.Time `json:"created_at"`
}
