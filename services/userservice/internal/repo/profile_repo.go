package repo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"messengermax/userservice/internal/entity"
)

var ErrNotFound = errors.New("record not found")

type ProfileRepository interface {
	Create(ctx context.Context, p *entity.Profile) error
	FindByID(ctx context.Context, id uint) (*entity.Profile, error)
	FindByIDs(ctx context.Context, ids []uint) ([]entity.Profile, error)
	FindByUsername(ctx context.Context, username string) (*entity.Profile, error)
	FindAll(ctx context.Context, excludeID uint) ([]entity.Profile, error)
	Search(ctx context.Context, query string, excludeID uint) ([]entity.Profile, error)
	Update(ctx context.Context, p *entity.Profile) error
	AddContact(ctx context.Context, userID, contactID uint) error
	RemoveContact(ctx context.Context, userID, contactID uint) error
	GetContacts(ctx context.Context, userID uint) ([]entity.Profile, error)
	IsContact(ctx context.Context, userID, contactID uint) (bool, error)
	Delete(ctx context.Context, id uint) error
}

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Create(ctx context.Context, p *entity.Profile) error {
	return r.db.Create(p).Error
}

func (r *profileRepository) FindByID(ctx context.Context, id uint) (*entity.Profile, error) {
	var p entity.Profile
	err := r.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (r *profileRepository) FindByUsername(ctx context.Context, username string) (*entity.Profile, error) {
	var p entity.Profile
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (r *profileRepository) FindByIDs(ctx context.Context, ids []uint) ([]entity.Profile, error) {
	var list []entity.Profile
	if len(ids) == 0 {
		return list, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
	return list, err
}

func (r *profileRepository) FindAll(ctx context.Context, excludeID uint) ([]entity.Profile, error) {
	var list []entity.Profile
	err := r.db.WithContext(ctx).Where("id <> ?", excludeID).Find(&list).Error
	return list, err
}

func (r *profileRepository) Search(ctx context.Context, query string, excludeID uint) ([]entity.Profile, error) {
	var list []entity.Profile
	like := "%" + query + "%"
	err := r.db.WithContext(ctx).Where("id <> ? AND (username ILIKE ? OR email ILIKE ?)", excludeID, like, like).Find(&list).Error
	return list, err
}

func (r *profileRepository) Update(ctx context.Context, p *entity.Profile) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *profileRepository) AddContact(ctx context.Context, userID, contactID uint) error {
	if userID == contactID {
		return errors.New("cannot add yourself")
	}
	return r.db.WithContext(ctx).Exec("INSERT INTO contacts (user_id, contact_id, created_at) VALUES (?, ?, NOW()) ON CONFLICT DO NOTHING", userID, contactID).Error
}

func (r *profileRepository) RemoveContact(ctx context.Context, userID, contactID uint) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM contacts WHERE user_id = ? AND contact_id = ?", userID, contactID).Error
}

func (r *profileRepository) GetContacts(ctx context.Context, userID uint) ([]entity.Profile, error) {
	var list []entity.Profile
	err := r.db.WithContext(ctx).
		Joins("JOIN contacts ON contacts.contact_id = profiles.id AND contacts.user_id = ?", userID).
		Find(&list).Error
	return list, err
}

func (r *profileRepository) IsContact(ctx context.Context, userID, contactID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("contacts").Where("user_id = ? AND contact_id = ?", userID, contactID).Count(&count).Error
	return count > 0, err
}

func (r *profileRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.Profile{}, id).Error
}
