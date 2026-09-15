package repo

import (
	"errors"

	"gorm.io/gorm"

	"messengermax/calls/internal/entity"
)

var ErrNotFound = errors.New("record not found")

type CallRepository interface {
	Create(c *entity.Call) error
	FindByID(id uint) (*entity.Call, error)
	FindActiveForUser(userID uint) (*entity.Call, error)
	FindExpiredRinging(nowMs int64) ([]entity.Call, error)
	Update(c *entity.Call) error
	History(userID uint, limit, offset int) ([]entity.Call, error)
}

type callRepository struct {
	db *gorm.DB
}

func NewCallRepository(db *gorm.DB) CallRepository {
	return &callRepository{db: db}
}

func (r *callRepository) Create(c *entity.Call) error {
	return r.db.Create(c).Error
}

func (r *callRepository) FindByID(id uint) (*entity.Call, error) {
	var c entity.Call
	err := r.db.First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (r *callRepository) FindActiveForUser(userID uint) (*entity.Call, error) {
	var c entity.Call
	err := r.db.
		Where("(caller_id = ? OR callee_id = ?) AND status IN (?)", userID, userID, []entity.CallStatus{
			entity.CallStatusRinging,
			entity.CallStatusActive,
		}).
		Order("id DESC").
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (r *callRepository) FindExpiredRinging(nowMs int64) ([]entity.Call, error) {
	var list []entity.Call
	err := r.db.
		Where("status = ? AND time_to_die_at > 0 AND time_to_die_at < ?", entity.CallStatusRinging, nowMs).
		Find(&list).Error
	return list, err
}

func (r *callRepository) Update(c *entity.Call) error {
	return r.db.Save(c).Error
}

func (r *callRepository) History(userID uint, limit, offset int) ([]entity.Call, error) {
	var list []entity.Call
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	err := r.db.
		Where("caller_id = ? OR callee_id = ?", userID, userID).
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&list).Error
	return list, err
}
