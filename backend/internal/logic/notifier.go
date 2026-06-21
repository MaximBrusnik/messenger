package logic

import "MessangerMax/internal/entity"

type Notifier interface {
	SendNewMessage(chatID uint, message *entity.MessageResponse)
	SendMessageEdited(chatID uint, message *entity.MessageResponse)
	SendReactionAdded(chatID uint, messageID uint, reaction *entity.ReactionResponse)
	SendReactionRemoved(chatID uint, messageID uint, userID uint, reaction string)
}
