package repo

import (
	"MessangerMax/internal/entity"
	"gorm.io/gorm"
)

type MusicRepository interface {
	Create(music *entity.Music) error
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

func (r *musicRepository) Create(music *entity.Music) error {
	return r.db.Create(music).Error
}

func (r *musicRepository) FindAll() ([]entity.Music, error) {
	var music []entity.Music
	err := r.db.Order("created_at DESC").Find(&music).Error
	return music, err
}

func (r *musicRepository) FindByStatus(status string) ([]entity.Music, error) {
	var music []entity.Music
	err := r.db.Where("status = ?", status).Order("created_at DESC").Find(&music).Error
	return music, err
}

func (r *musicRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&entity.Music{}).Where("id = ?", id).Update("status", status).Error
}

func (r *musicRepository) FindByID(id uint) (*entity.Music, error) {
	var music entity.Music
	err := r.db.First(&music, id).Error
	return &music, err
}

func (r *musicRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Music{}, id).Error
}

func (r *musicRepository) GetTotalSize() (int64, error) {
	var sum struct {
		Total int64
	}
	err := r.db.Model(&entity.Music{}).Select("COALESCE(SUM(size), 0) AS total").Scan(&sum).Error
	return sum.Total, err
}
