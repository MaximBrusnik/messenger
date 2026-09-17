package repo

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"messengermax/chatservice/internal/entity"
)

var ErrNotFound = errors.New("record not found")

type ChatRepository interface {
	Create(chat *entity.Chat, participantIDs []uint) error
	FindByID(id uint) (*entity.Chat, error)
	FindByUserID(userID uint) ([]entity.Chat, error)
	FindByUserIDArchived(userID uint) ([]entity.Chat, error)
	FindPrivateChat(userID1, userID2 uint) (*entity.Chat, error)
	FindFavoritesChat(userID uint) (*entity.Chat, error)
	GetParticipantIDs(chatID uint) ([]uint, error)
	GetCommonChatIDs(userID1, userID2 uint) ([]uint, error)
	UpdateLastRead(chatID, userID uint) error
	GetUnreadCount(chatID, userID uint) (int, error)
	AddUserToChat(chatID, userID uint) error
	RemoveUserFromChat(chatID, userID uint) error
	IsParticipant(chatID, userID uint) (bool, error)
	ArchiveChat(chatID, userID uint) error
	UnarchiveChat(chatID, userID uint) error
	PinMessage(chatID, messageID uint) error
	UnpinMessage(chatID uint) error
	Delete(chatID uint) error
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) Create(chat *entity.Chat, participantIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(chat).Error; err != nil {
			return err
		}
		for _, uid := range participantIDs {
			cu := entity.ChatUser{ChatID: chat.ID, UserID: uid, JoinedAt: time.Now()}
			if err := tx.Create(&cu).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *chatRepository) FindByID(id uint) (*entity.Chat, error) {
	var c entity.Chat
	err := r.db.First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (r *chatRepository) FindByUserID(userID uint) ([]entity.Chat, error) {
	var chats []entity.Chat
	err := r.db.
		Joins("JOIN chat_users ON chat_users.chat_id = chats.id AND chat_users.user_id = ?", userID).
		Where("chats.deleted_at IS NULL AND chat_users.is_archived = ?", false).
		Order("chats.updated_at DESC").
		Find(&chats).Error
	return chats, err
}

func (r *chatRepository) FindByUserIDArchived(userID uint) ([]entity.Chat, error) {
	var chats []entity.Chat
	err := r.db.
		Joins("JOIN chat_users ON chat_users.chat_id = chats.id AND chat_users.user_id = ?", userID).
		Where("chats.deleted_at IS NULL AND chat_users.is_archived = ?", true).
		Order("chats.updated_at DESC").
		Find(&chats).Error
	return chats, err
}

func (r *chatRepository) FindPrivateChat(userID1, userID2 uint) (*entity.Chat, error) {
	var chat entity.Chat
	err := r.db.Raw(`
		SELECT c.* FROM chats c
		WHERE c.type = 'private' AND c.deleted_at IS NULL
		AND EXISTS (SELECT 1 FROM chat_users a WHERE a.chat_id = c.id AND a.user_id = ?)
		AND EXISTS (SELECT 1 FROM chat_users b WHERE b.chat_id = c.id AND b.user_id = ?)
	`, userID1, userID2).Scan(&chat).Error
	if err != nil {
		return nil, err
	}
	if chat.ID == 0 {
		return nil, ErrNotFound
	}
	return &chat, nil
}

func (r *chatRepository) FindFavoritesChat(userID uint) (*entity.Chat, error) {
	var chat entity.Chat
	err := r.db.Raw(`
		SELECT c.* FROM chats c
		WHERE c.type = 'private' AND c.deleted_at IS NULL
		AND EXISTS (
			SELECT 1 FROM chat_users a
			WHERE a.chat_id = c.id AND a.user_id = ?
		)
		AND NOT EXISTS (
			SELECT 1 FROM chat_users b
			WHERE b.chat_id = c.id AND b.user_id <> ?
		)
	`, userID, userID).Scan(&chat).Error
	if err != nil {
		return nil, err
	}
	if chat.ID == 0 {
		return nil, ErrNotFound
	}
	return &chat, nil
}

func (r *chatRepository) GetParticipantIDs(chatID uint) ([]uint, error) {
	var ids []uint
	err := r.db.Table("chat_users").Where("chat_id = ?", chatID).Pluck("user_id", &ids).Error
	return ids, err
}

func (r *chatRepository) GetCommonChatIDs(userID1, userID2 uint) ([]uint, error) {
	var ids []uint
	err := r.db.Raw(`
		SELECT a.chat_id FROM chat_users a
		JOIN chat_users b ON b.chat_id = a.chat_id
		JOIN chats c ON c.id = a.chat_id
		WHERE a.user_id = ? AND b.user_id = ? AND c.deleted_at IS NULL
	`, userID1, userID2).Pluck("chat_id", &ids).Error
	return ids, err
}

func (r *chatRepository) UpdateLastRead(chatID, userID uint) error {
	return r.db.Exec("UPDATE chat_users SET last_read = ? WHERE chat_id = ? AND user_id = ?", time.Now(), chatID, userID).Error
}

func (r *chatRepository) GetUnreadCount(chatID, userID uint) (int, error) {
	var count int64
	err := r.db.Raw(`
		SELECT COUNT(*) FROM messages m
		JOIN chat_users cu ON cu.chat_id = m.chat_id AND cu.user_id = ?
		WHERE m.chat_id = ? AND m.sender_id <> ? AND m.created_at > cu.last_read AND m.deleted_at IS NULL
	`, userID, chatID, userID).Scan(&count).Error
	return int(count), err
}

func (r *chatRepository) AddUserToChat(chatID, userID uint) error {
	cu := entity.ChatUser{ChatID: chatID, UserID: userID, JoinedAt: time.Now()}
	return r.db.Create(&cu).Error
}

func (r *chatRepository) RemoveUserFromChat(chatID, userID uint) error {
	return r.db.Where("chat_id = ? AND user_id = ?", chatID, userID).Delete(&entity.ChatUser{}).Error
}

func (r *chatRepository) IsParticipant(chatID, userID uint) (bool, error) {
	var count int64
	err := r.db.Table("chat_users").Where("chat_id = ? AND user_id = ?", chatID, userID).Count(&count).Error
	return count > 0, err
}

func (r *chatRepository) ArchiveChat(chatID, userID uint) error {
	return r.db.Table("chat_users").Where("chat_id = ? AND user_id = ?", chatID, userID).Update("is_archived", true).Error
}

func (r *chatRepository) UnarchiveChat(chatID, userID uint) error {
	return r.db.Table("chat_users").Where("chat_id = ? AND user_id = ?", chatID, userID).Update("is_archived", false).Error
}

func (r *chatRepository) PinMessage(chatID, messageID uint) error {
	return r.db.Model(&entity.Chat{}).Where("id = ?", chatID).Update("pinned_message_id", messageID).Error
}

func (r *chatRepository) UnpinMessage(chatID uint) error {
	return r.db.Model(&entity.Chat{}).Where("id = ?", chatID).Update("pinned_message_id", nil).Error
}

func (r *chatRepository) Delete(chatID uint) error {
	return r.db.Delete(&entity.Chat{}, chatID).Error
}
