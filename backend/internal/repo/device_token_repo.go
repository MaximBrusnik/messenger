package repo

import (
	"MessangerMax/internal/entity"
	"gorm.io/gorm"
)

type DeviceTokenRepository interface {
	Create(token *entity.DeviceToken) error
	FindByUserID(userID uint) ([]entity.DeviceToken, error)
	DeleteByToken(token string) error
}

type deviceTokenRepository struct {
	db *gorm.DB
}

func NewDeviceTokenRepository(db *gorm.DB) DeviceTokenRepository {
	return &deviceTokenRepository{db: db}
}

func (r *deviceTokenRepository) Create(token *entity.DeviceToken) error {
	return r.db.Create(token).Error
}

func (r *deviceTokenRepository) FindByUserID(userID uint) ([]entity.DeviceToken, error) {
	var tokens []entity.DeviceToken
	err := r.db.Where("user_id = ?", userID).Find(&tokens).Error
	return tokens, err
}

func (r *deviceTokenRepository) DeleteByToken(token string) error {
	return r.db.Where("token = ?", token).Delete(&entity.DeviceToken{}).Error
}
