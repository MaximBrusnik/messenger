package repo

import (
	"errors"

	"gorm.io/gorm"

	"messengermax/music/internal/entity"
)

var ErrNotFound = errors.New("record not found")

type MusicRepository interface {
	Create(m *entity.Music) error
	FindAll() ([]entity.Music, error)
	FindByID(id uint) (*entity.Music, error)
	FindByStatus(status string) ([]entity.Music, error)
	UpdateStatus(id uint, status string) error
	Delete(id uint) error
	GetTotalSize() (int64, error)
}

type musicRepository struct {
	db *gorm.DB
}

func NewMusicRepository(db *gorm.DB) MusicRepository {
	return &musicRepository{db: db}
}

func (r *musicRepository) Create(m *entity.Music) error {
	return r.db.Create(m).Error
}

func (r *musicRepository) FindAll() ([]entity.Music, error) {
	var list []entity.Music
	err := r.db.Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *musicRepository) FindByID(id uint) (*entity.Music, error) {
	var m entity.Music
	err := r.db.First(&m, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (r *musicRepository) FindByStatus(status string) ([]entity.Music, error) {
	var list []entity.Music
	err := r.db.Where("status = ?", status).Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *musicRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&entity.Music{}).Where("id = ?", id).Update("status", status).Error
}

func (r *musicRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Music{}, id).Error
}

func (r *musicRepository) GetTotalSize() (int64, error) {
	var total int64
	err := r.db.Model(&entity.Music{}).Select("COALESCE(SUM(size),0)").Scan(&total).Error
	return total, err
}
