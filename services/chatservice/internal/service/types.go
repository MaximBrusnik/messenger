package service

import (
	"time"

	"messengermax/chatservice/internal/entity"
)

type ChatResponse struct {
	ID              uint
	Name            string
	Type            string
	Avatar          string
	PinnedMessageID *uint
	Participants    []uint
	UnreadCount     int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	IsFavorites     bool
	LastMessage     *MessageWithReactions
	PinnedMessage   *MessageWithReactions
}

type MessageWithReactions struct {
	Msg       *entity.Message
	Reactions []entity.MessageReaction
	ReplyTo   *entity.Message
}
