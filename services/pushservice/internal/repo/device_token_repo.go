package repo

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"messengermax/pushservice/internal/entity"
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
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "token"}},
		Where: clause.Where{
			Exprs: []clause.Expression{clause.Expr{SQL: "device_tokens.deleted_at IS NULL"}},
		},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "platform", "deleted_at", "updated_at"}),
	}).Create(token).Error
}

func (r *deviceTokenRepository) DeleteByToken(token string) error {
	return r.db.Where("token = ?", token).Delete(&entity.DeviceToken{}).Error
}

func (r *deviceTokenRepository) FindByUserID(userID uint) ([]entity.DeviceToken, error) {
	var list []entity.DeviceToken
	err := r.db.Where("user_id = ?", userID).Find(&list).Error
	return list, err
}
