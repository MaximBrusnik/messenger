package repo

import (
	"errors"

	"gorm.io/gorm"

	"messengermax/userservice/internal/entity"
)

var ErrNotFound = errors.New("record not found")

type ProfileRepository interface {
	Create(p *entity.Profile) error
	FindByID(id uint) (*entity.Profile, error)
	FindByIDs(ids []uint) ([]entity.Profile, error)
	FindByUsername(username string) (*entity.Profile, error)
	FindAll(excludeID uint) ([]entity.Profile, error)
	Search(query string, excludeID uint) ([]entity.Profile, error)
	Update(p *entity.Profile) error
	AddContact(userID, contactID uint) error
	RemoveContact(userID, contactID uint) error
	GetContacts(userID uint) ([]entity.Profile, error)
	IsContact(userID, contactID uint) (bool, error)
}

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Create(p *entity.Profile) error {
	return r.db.Create(p).Error
}

func (r *profileRepository) FindByID(id uint) (*entity.Profile, error) {
	var p entity.Profile
	err := r.db.First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (r *profileRepository) FindByUsername(username string) (*entity.Profile, error) {
	var p entity.Profile
	err := r.db.Where("username = ?", username).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (r *profileRepository) FindByIDs(ids []uint) ([]entity.Profile, error) {
	var list []entity.Profile
	if len(ids) == 0 {
		return list, nil
	}
	err := r.db.Where("id IN ?", ids).Find(&list).Error
	return list, err
}

func (r *profileRepository) FindAll(excludeID uint) ([]entity.Profile, error) {
	var list []entity.Profile
	err := r.db.Where("id <> ?", excludeID).Find(&list).Error
	return list, err
}

func (r *profileRepository) Search(query string, excludeID uint) ([]entity.Profile, error) {
	var list []entity.Profile
	like := "%" + query + "%"
	err := r.db.Where("id <> ? AND (username ILIKE ? OR email ILIKE ?)", excludeID, like, like).Find(&list).Error
	return list, err
}

func (r *profileRepository) Update(p *entity.Profile) error {
	return r.db.Save(p).Error
}

func (r *profileRepository) AddContact(userID, contactID uint) error {
	if userID == contactID {
		return errors.New("cannot add yourself")
	}
	return r.db.Exec("INSERT INTO contacts (user_id, contact_id, created_at) VALUES (?, ?, NOW()) ON CONFLICT DO NOTHING", userID, contactID).Error
}

func (r *profileRepository) RemoveContact(userID, contactID uint) error {
	return r.db.Exec("DELETE FROM contacts WHERE user_id = ? AND contact_id = ?", userID, contactID).Error
}

func (r *profileRepository) GetContacts(userID uint) ([]entity.Profile, error) {
	var list []entity.Profile
	err := r.db.
		Joins("JOIN contacts ON contacts.contact_id = profiles.id AND contacts.user_id = ?", userID).
		Find(&list).Error
	return list, err
}

func (r *profileRepository) IsContact(userID, contactID uint) (bool, error) {
	var count int64
	err := r.db.Table("contacts").Where("user_id = ? AND contact_id = ?", userID, contactID).Count(&count).Error
	return count > 0, err
}
