package repo

import (
	"errors"

	"gorm.io/gorm"

	"messengermax/authservice/internal/entity"
)

var ErrNotFound = errors.New("record not found")

type UserRepository interface {
	Create(user *entity.User) error
	FindByID(id uint) (*entity.User, error)
	FindByUsername(username string) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
	FindByVerificationToken(token string) (*entity.User, error)
	Update(user *entity.User) error
	UpdateLastLogin(userID uint, t interface{}) error
	UpdatePassword(userID uint, hashed string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *entity.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByID(id uint) (*entity.User, error) {
	var u entity.User
	err := r.db.First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *userRepository) FindByUsername(username string) (*entity.User, error) {
	var u entity.User
	err := r.db.Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *userRepository) FindByEmail(email string) (*entity.User, error) {
	var u entity.User
	err := r.db.Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *userRepository) FindByVerificationToken(token string) (*entity.User, error) {
	var u entity.User
	err := r.db.Where("verification_token = ?", token).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *userRepository) Update(user *entity.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) UpdateLastLogin(userID uint, t interface{}) error {
	return r.db.Model(&entity.User{}).Where("id = ?", userID).Update("last_login", t).Error
}

func (r *userRepository) UpdatePassword(userID uint, hashed string) error {
	return r.db.Model(&entity.User{}).Where("id = ?", userID).Update("password", hashed).Error
}
