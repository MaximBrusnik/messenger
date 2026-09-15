package repo

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"messengermax/chat/internal/entity"
)

type MessageRepository interface {
	Create(message *entity.Message) error
	FindByID(id uint) (*entity.Message, error)
	FindByChatID(chatID uint, limit, offset int) ([]entity.Message, error)
	MarkAsRead(messageID uint) error
	MarkChatAsRead(chatID, userID uint) ([]uint, error)
	GetLastMessage(chatID uint) (*entity.Message, error)
	UpdateText(messageID uint, text string) error
	Delete(messageID uint) error
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(message *entity.Message) error {
	return r.db.Create(message).Error
}

func (r *messageRepository) FindByID(id uint) (*entity.Message, error) {
	var m entity.Message
	err := r.db.First(&m, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (r *messageRepository) FindByChatID(chatID uint, limit, offset int) ([]entity.Message, error) {
	var messages []entity.Message
	err := r.db.Where("chat_id = ?", chatID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&messages).Error
	// reverse to chronological order (oldest first) for display
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, err
}

func (r *messageRepository) MarkAsRead(messageID uint) error {
	return r.db.Model(&entity.Message{}).Where("id = ?", messageID).
		Updates(map[string]interface{}{"is_read": true, "read_at": time.Now()}).Error
}

func (r *messageRepository) MarkChatAsRead(chatID, userID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Raw(`
		SELECT id FROM messages
		WHERE chat_id = ? AND sender_id <> ? AND is_read = false AND deleted_at IS NULL
	`, chatID, userID).Pluck("id", &ids).Error
	if err != nil || len(ids) == 0 {
		return ids, err
	}
	err = r.db.Model(&entity.Message{}).
		Where("chat_id = ? AND sender_id <> ? AND is_read = false", chatID, userID).
		Updates(map[string]interface{}{"is_read": true, "read_at": time.Now()}).Error
	return ids, err
}

func (r *messageRepository) GetLastMessage(chatID uint) (*entity.Message, error) {
	var m entity.Message
	err := r.db.Where("chat_id = ?", chatID).Order("created_at DESC").First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &m, err
}

func (r *messageRepository) UpdateText(messageID uint, text string) error {
	return r.db.Model(&entity.Message{}).Where("id = ?", messageID).
		Updates(map[string]interface{}{"text": text, "edited": true, "edited_at": time.Now()}).Error
}

func (r *messageRepository) Delete(messageID uint) error {
	return r.db.Delete(&entity.Message{}, messageID).Error
}
