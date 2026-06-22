package repo

import (
	"MessangerMax/internal/entity"
	"errors"
	"gorm.io/gorm"
	"time"
)

type ChatRepository interface {
	Create(chat *entity.Chat) error
	FindByID(id uint) (*entity.Chat, error)
	FindByUserID(userID uint) ([]entity.Chat, error)
	AddUserToChat(chatID, userID uint) error
	RemoveUserFromChat(chatID, userID uint) error
	UpdateLastRead(chatID, userID uint) error
	GetUnreadCount(chatID, userID uint) (int, error)
	FindPrivateChat(userID1, userID2 uint) (*entity.Chat, error)
	GetParticipantIDs(chatID uint) ([]uint, error)
	GetCommonChatIDs(userID1, userID2 uint) ([]uint, error)
	Delete(chatID uint) error
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) Create(chat *entity.Chat) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(chat).Error; err != nil {
			return err
		}
		// Добавляем участников
		for _, user := range chat.Participants {
			if err := tx.Exec(`INSERT INTO chat_users (chat_id, user_id, joined_at, is_admin) VALUES (?, ?, ?, ?)                 ON CONFLICT (chat_id, user_id) DO NOTHING
`, chat.ID, user.ID, time.Now(), false).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *chatRepository) FindByID(id uint) (*entity.Chat, error) {
	var chat entity.Chat
	err := r.db.Preload("Participants").First(&chat, id).Error
	return &chat, err
}

func (r *chatRepository) FindByUserID(userID uint) ([]entity.Chat, error) {
	var chats []entity.Chat

	err := r.db.Raw(`
        SELECT c.* FROM chats c
        JOIN chat_users cu ON c.id = cu.chat_id
        WHERE cu.user_id = ? AND c.deleted_at IS NULL
        ORDER BY c.updated_at DESC
    `, userID).Scan(&chats).Error

	if err != nil {
		return nil, err
	}

	// Загружаем участников и последнее сообщение для каждого чата
	for i := range chats {
		r.db.Model(&chats[i]).Association("Participants").Find(&chats[i].Participants)
	}

	return chats, nil
}

func (r *chatRepository) AddUserToChat(chatID, userID uint) error {
	return r.db.Exec(`
        INSERT INTO chat_users (chat_id, user_id, joined_at, is_admin) 
        VALUES (?, ?, ?, ?)
    `, chatID, userID, time.Now(), false).Error
}

func (r *chatRepository) RemoveUserFromChat(chatID, userID uint) error {
	return r.db.Exec(`
        DELETE FROM chat_users WHERE chat_id = ? AND user_id = ?
    `, chatID, userID).Error
}

func (r *chatRepository) UpdateLastRead(chatID, userID uint) error {
	return r.db.Exec(`
        UPDATE chat_users SET last_read = ? 
        WHERE chat_id = ? AND user_id = ?
    `, time.Now(), chatID, userID).Error
}

func (r *chatRepository) GetUnreadCount(chatID, userID uint) (int, error) {
	var count int64

	err := r.db.Raw(`
        SELECT COUNT(*) FROM messages m
        WHERE m.chat_id = ? 
        AND m.sender_id != ?
        AND m.created_at > COALESCE(
            (SELECT last_read FROM chat_users WHERE chat_id = ? AND user_id = ?), 
            '1970-01-01'
        )
    `, chatID, userID, chatID, userID).Count(&count).Error

	return int(count), err
}

func (r *chatRepository) GetParticipantIDs(chatID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Table("chat_users").
		Where("chat_id = ?", chatID).
		Pluck("user_id", &ids).Error
	return ids, err
}

func (r *chatRepository) GetCommonChatIDs(userID1, userID2 uint) ([]uint, error) {
	var ids []uint
	err := r.db.Raw(`
		SELECT cu1.chat_id FROM chat_users cu1
		JOIN chat_users cu2 ON cu1.chat_id = cu2.chat_id
		JOIN chats c ON c.id = cu1.chat_id
		WHERE cu1.user_id = ? AND cu2.user_id = ?
		AND c.deleted_at IS NULL
	`, userID1, userID2).Pluck("chat_id", &ids).Error
	return ids, err
}

func (r *chatRepository) Delete(chatID uint) error {
	return r.db.Delete(&entity.Chat{}, chatID).Error
}

func (r *chatRepository) FindPrivateChat(userID1, userID2 uint) (*entity.Chat, error) {
	var chat entity.Chat

	err := r.db.Raw(`
        SELECT c.* FROM chats c
        JOIN chat_users cu1 ON c.id = cu1.chat_id
        JOIN chat_users cu2 ON c.id = cu2.chat_id
        WHERE c.type = 'private'
        AND cu1.user_id = ?
        AND cu2.user_id = ?
        AND c.deleted_at IS NULL
        LIMIT 1
    `, userID1, userID2).Scan(&chat).Error

	if err != nil {
		return nil, err
	}

	if chat.ID == 0 {
		return nil, errors.New("чат не найден")
	}

	return &chat, nil
}
