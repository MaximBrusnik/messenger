package entity

import "time"

type MessageReaction struct {
	MessageID uint      `gorm:"primaryKey" json:"message_id"`
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	Reaction  string    `gorm:"primaryKey;size:50" json:"reaction"`
	CreatedAt time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type AddReactionRequest struct {
	Reaction string `json:"reaction" binding:"required"`
}

type ReactionResponse struct {
	MessageID uint      `json:"message_id"`
	UserID    uint      `json:"user_id"`
	Reaction  string    `json:"reaction"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username,omitempty"`
}
