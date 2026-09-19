package repo

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"messengermax/authservice/internal/entity"
)

var ErrNotFound = errors.New("record not found")

type UserRepository interface {
	Create(user *entity.User) error
	CountUsers() (int64, error)
	FindByID(id uint) (*entity.User, error)
	FindByUsername(username string) (*entity.User, error)
	FindByEmail(email string) (*entity.User, error)
	FindByVerificationToken(token string) (*entity.User, error)
	Update(user *entity.User) error
	UpdateLastLogin(userID uint, t interface{}) error
	UpdatePassword(userID uint, hashed string) error
	CreateSession(s *entity.Session) error
	TouchSession(jti string) error
	ListSessions(userID uint) ([]entity.Session, error)
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

func (r *userRepository) CountUsers() (int64, error) {
	var n int64
	err := r.db.Model(&entity.User{}).Where("is_bot = ?", false).Count(&n).Error
	return n, err
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

func (r *userRepository) CreateSession(s *entity.Session) error {
	return r.db.Create(s).Error
}

func (r *userRepository) TouchSession(jti string) error {
	return r.db.Model(&entity.Session{}).Where("id = ?", jti).
		Update("last_login_at", time.Now()).Error
}

func (r *userRepository) ListSessions(userID uint) ([]entity.Session, error) {
	var sessions []entity.Session
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *userRepository) DeleteSessions(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&entity.Session{}).Error
}

func (r *userRepository) Delete(userID uint) error {
	return r.db.Delete(&entity.User{}, userID).Error
}

func (r *userRepository) ListAdmin(f AdminListFilter) ([]entity.User, int64, error) {
	q := r.db.Model(&entity.User{})
	if f.Query != "" {
		like := "%" + f.Query + "%"
		q = q.Where("username ILIKE ? OR email ILIKE ?", like, like)
	}
	if f.IsBot == nil {
		q = q.Where("is_bot = ?", false)
	} else {
		q = q.Where("is_bot = ?", *f.IsBot)
	}
	if f.IsAdmin != nil {
		q = q.Where("is_admin = ?", *f.IsAdmin)
	}
	if f.Active != nil {
		q = q.Where("is_active = ?", *f.Active)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := f.Page
	if page == 0 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize == 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	var list []entity.User
	err := q.Order("created_at DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&list).Error
	return list, total, err
}

func (r *userRepository) CountStats(since time.Time) (UserCounts, error) {
	var c UserCounts
	err := r.db.Model(&entity.User{}).
		Where("is_bot = ?", false).Count(&c.TotalUsers).Error
	if err != nil {
		return c, err
	}
	if err := r.db.Model(&entity.User{}).
		Where("is_bot = ? AND is_active = ?", false, true).Count(&c.ActiveUsers).Error; err != nil {
		return c, err
	}
	if err := r.db.Model(&entity.User{}).
		Where("is_bot = ? AND is_active = ?", false, false).Count(&c.BannedUsers).Error; err != nil {
		return c, err
	}
	if err := r.db.Model(&entity.User{}).
		Where("is_bot = ? AND email_verified = ?", false, false).Count(&c.UnverifiedUsers).Error; err != nil {
		return c, err
	}
	if err := r.db.Model(&entity.User{}).
		Where("is_bot = ?", true).Count(&c.Bots).Error; err != nil {
		return c, err
	}
	if err := r.db.Model(&entity.User{}).
		Where("is_admin = ?", true).Count(&c.Admins).Error; err != nil {
		return c, err
	}
	if err := r.db.Model(&entity.User{}).
		Where("is_bot = ? AND created_at >= ?", false, since).Count(&c.NewLast7Days).Error; err != nil {
		return c, err
	}
	return c, nil
}
