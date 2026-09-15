package repo

import (
	"gorm.io/gorm"

	"messengermax/push/internal/entity"
)

type DeviceTokenRepository interface {
	Add(token *entity.DeviceToken) error
	DeleteByToken(token string) error
	FindByUserID(userID uint) ([]entity.DeviceToken, error)
}

type deviceTokenRepository struct {
	db *gorm.DB
}

func NewDeviceTokenRepository(db *gorm.DB) DeviceTokenRepository {
	return &deviceTokenRepository{db: db}
}

func (r *deviceTokenRepository) Add(token *entity.DeviceToken) error {
	return r.db.Create(token).Error
}

func (r *deviceTokenRepository) DeleteByToken(token string) error {
	return r.db.Where("token = ?", token).Delete(&entity.DeviceToken{}).Error
}

func (r *deviceTokenRepository) FindByUserID(userID uint) ([]entity.DeviceToken, error) {
	var list []entity.DeviceToken
	err := r.db.Where("user_id = ?", userID).Find(&list).Error
	return list, err
}
