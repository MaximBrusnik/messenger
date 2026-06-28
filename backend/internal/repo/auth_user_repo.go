package repo

import (
	"MessangerMax/internal/entity"
	"errors"
	"gorm.io/gorm"
	"time"
)

type UserRepository interface {
	Create(user *entity.User) error
	FindByID(id uint) (*entity.User, error)
	FindByUsername(username string) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
	FindByVerificationToken(token string) (*entity.User, error)
	Update(user *entity.User) error
	FindAll(excludeID uint) ([]entity.User, error)
	Search(query string, excludeID uint) ([]entity.User, error)
	AddContact(userID, contactID uint) error
	RemoveContact(userID, contactID uint) error
	GetContacts(userID uint) ([]entity.User, error)
	IsContact(userID, contactID uint) (bool, error)
	UpdatePassword(userID uint, hashedPassword string) error
	UpdateLastLogin(userID uint, t time.Time) error
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
	var user entity.User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *userRepository) FindByUsername(username string) (*entity.User, error) {
	var user entity.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, err
}

func (r *userRepository) FindByEmail(email string) (*entity.User, error) {
	var user entity.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, err
}

func (r *userRepository) FindByVerificationToken(token string) (*entity.User, error) {
	var user entity.User
	err := r.db.Where("verification_token = ?", token).First(&user).Error
	return &user, err
}

func (r *userRepository) Update(user *entity.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) FindAll(excludeID uint) ([]entity.User, error) {
	var users []entity.User
	err := r.db.Where("id != ?", excludeID).
		Select("id, username, email, created_at, last_login, avatar, status").
		Find(&users).Error
	return users, err
}

func (r *userRepository) Search(query string, excludeID uint) ([]entity.User, error) {
	var users []entity.User
	err := r.db.Where("id != ? AND (username LIKE ? OR email LIKE ?)",
		excludeID,
		"%"+query+"%",
		"%"+query+"%").
		Select("id, username, email, created_at, last_login, avatar, status").
		Find(&users).Error
	return users, err
}

func (r *userRepository) AddContact(userID, contactID uint) error {
	// Проверяем, не является ли пользователь сам себе контактом
	if userID == contactID {
		return errors.New("нельзя добавить себя в контакты")
	}

	// Проверяем, существует ли уже такой контакт
	var count int64
	r.db.Table("user_contacts").
		Where("user_id = ? AND contact_id = ?", userID, contactID).
		Count(&count)

	if count > 0 {
		return errors.New("контакт уже существует")
	}

	return r.db.Exec(`
        INSERT INTO user_contacts (user_id, contact_id) 
        VALUES (?, ?)
    `, userID, contactID).Error
}

func (r *userRepository) RemoveContact(userID, contactID uint) error {
	return r.db.Exec(`
        DELETE FROM user_contacts 
        WHERE user_id = ? AND contact_id = ?
    `, userID, contactID).Error
}

func (r *userRepository) GetContacts(userID uint) ([]entity.User, error) {
	var contacts []entity.User
	err := r.db.Raw(`
		SELECT u.id, u.username, u.email, u.created_at, u.last_login, u.avatar, u.status 
		FROM users u
        JOIN user_contacts uc ON u.id = uc.contact_id
        WHERE uc.user_id = ?
        ORDER BY u.username
    `, userID).Scan(&contacts).Error
	return contacts, err
}

func (r *userRepository) IsContact(userID, contactID uint) (bool, error) {
	var count int64
	err := r.db.Table("user_contacts").
		Where("user_id = ? AND contact_id = ?", userID, contactID).
		Count(&count).Error
	return count > 0, err
}

func (r *userRepository) UpdatePassword(userID uint, hashedPassword string) error {
	return r.db.Model(&entity.User{}).
		Where("id = ?", userID).
		Update("password", hashedPassword).Error
}

func (r *userRepository) UpdateLastLogin(userID uint, t time.Time) error {
	return r.db.Model(&entity.User{}).
		Where("id = ?", userID).
		Update("last_login", t).Error
}
