package repo

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"messengermax/chat/internal/entity"
)

type ReactionRepository interface {
	Add(reaction *entity.MessageReaction) error
	Remove(messageID, userID uint, reaction string) error
	FindByMessageID(messageID uint) ([]entity.MessageReaction, error)
}

type reactionRepository struct {
	db *gorm.DB
}

func NewReactionRepository(db *gorm.DB) ReactionRepository {
	return &reactionRepository{db: db}
}

func (r *reactionRepository) Add(reaction *entity.MessageReaction) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(reaction).Error
}

func (r *reactionRepository) Remove(messageID, userID uint, reaction string) error {
	return r.db.
		Where("message_id = ? AND user_id = ? AND reaction = ?", messageID, userID, reaction).
		Delete(&entity.MessageReaction{}).Error
}

func (r *reactionRepository) FindByMessageID(messageID uint) ([]entity.MessageReaction, error) {
	var list []entity.MessageReaction
	err := r.db.Where("message_id = ?", messageID).Find(&list).Error
	return list, err
}
