package logic

import "MessangerMax/internal/entity"

type Notifier interface {
	SendNewMessage(chatID uint, message *entity.MessageResponse)
	SendMessageEdited(chatID uint, message *entity.MessageResponse)
	SendReactionAdded(chatID uint, messageID uint, reaction *entity.ReactionResponse)
	SendReactionRemoved(chatID uint, messageID uint, userID uint, reaction string)
	SendMessageDeleted(chatID uint, messageID uint)
	SendChatDeleted(chatID uint)
	SendMessagesRead(chatID uint, messageIDs []uint, readByUserID uint)
}
