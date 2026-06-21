package repo

import (
	"MessangerMax/internal/entity"
	"gorm.io/gorm"
	"time"
)

type MessageRepository interface {
	Create(message *entity.Message) error
	FindByID(id uint) (*entity.Message, error)
	FindByChatID(chatID uint, limit, offset int) ([]entity.Message, error)
	MarkAsRead(messageID uint) error
	MarkChatAsRead(chatID, userID uint) error
	GetLastMessage(chatID uint) (*entity.Message, error)
	UpdateText(messageID uint, text string) error
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
	var message entity.Message
	err := r.db.Preload("Sender").
		Preload("Reactions").
		First(&message, id).Error
	return &message, err
}

func (r *messageRepository) FindByChatID(chatID uint, limit, offset int) ([]entity.Message, error) {
	var messages []entity.Message

	query := r.db.Where("chat_id = ?", chatID).
		Preload("Sender").
		Preload("Reactions").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&messages).Error

	// Переворачиваем порядок для хронологического отображения
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, err
}

func (r *messageRepository) MarkAsRead(messageID uint) error {
	return r.db.Model(&entity.Message{}).
		Where("id = ?", messageID).
		Update("is_read", true).Error
}

func (r *messageRepository) MarkChatAsRead(chatID, userID uint) error {
	return r.db.Exec(`
        UPDATE messages m
        SET is_read = true
        WHERE m.chat_id = ? 
        AND m.sender_id != ?
        AND m.is_read = false
    `, chatID, userID).Error
}

func (r *messageRepository) UpdateText(messageID uint, text string) error {
	now := time.Now()
	return r.db.Model(&entity.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"text":      text,
			"edited":    true,
			"edited_at": now,
		}).Error
}

func (r *messageRepository) GetLastMessage(chatID uint) (*entity.Message, error) {
	var message entity.Message
	err := r.db.Where("chat_id = ?", chatID).
		Order("created_at DESC").
		First(&message).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	return &message, err
}
