package repo

import (
	"MessangerMax/internal/entity"
	"gorm.io/gorm"
)

type ReactionRepository interface {
	Add(reaction *entity.MessageReaction) error
	Remove(messageID, userID uint, reaction string) error
	FindByMessageID(messageID uint) ([]entity.MessageReaction, error)
	FindByMessageAndUser(messageID, userID uint) ([]entity.MessageReaction, error)
}

type reactionRepository struct {
	db *gorm.DB
}

func NewReactionRepository(db *gorm.DB) ReactionRepository {
	return &reactionRepository{db: db}
}

func (r *reactionRepository) Add(reaction *entity.MessageReaction) error {
	return r.db.Create(reaction).Error
}

func (r *reactionRepository) Remove(messageID, userID uint, reaction string) error {
	return r.db.Where("message_id = ? AND user_id = ? AND reaction = ?", messageID, userID, reaction).
		Delete(&entity.MessageReaction{}).Error
}

func (r *reactionRepository) FindByMessageID(messageID uint) ([]entity.MessageReaction, error) {
	var reactions []entity.MessageReaction
	err := r.db.Preload("User").
		Where("message_id = ?", messageID).
		Find(&reactions).Error
	return reactions, err
}

func (r *reactionRepository) FindByMessageAndUser(messageID, userID uint) ([]entity.MessageReaction, error) {
	var reactions []entity.MessageReaction
	err := r.db.Where("message_id = ? AND user_id = ?", messageID, userID).
		Find(&reactions).Error
	return reactions, err
}
