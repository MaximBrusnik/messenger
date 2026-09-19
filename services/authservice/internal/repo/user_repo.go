package repo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"messengermax/authservice/internal/entity"
)

var ErrNotFound = errors.New("record not found")

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	CountUsers(ctx context.Context) (int64, error)
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByVerificationToken(ctx context.Context, token string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	UpdateLastLogin(ctx context.Context, userID uint, t interface{}) error
	UpdatePassword(ctx context.Context, userID uint, hashed string) error
	CreateSession(ctx context.Context, s *entity.Session) error
	TouchSession(ctx context.Context, jti string) error
	ListSessions(ctx context.Context, userID uint) ([]entity.Session, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.User{}).Where("is_bot = ?", false).Count(&n).Error
	return n, err
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var u entity.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var u entity.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *userRepository) FindByVerificationToken(ctx context.Context, token string) (*entity.User, error) {
	var u entity.User
	err := r.db.WithContext(ctx).Where("verification_token = ?", token).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, userID uint, t interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", userID).Update("last_login", t).Error
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID uint, hashed string) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", userID).Update("password", hashed).Error
}

func (r *userRepository) CreateSession(ctx context.Context, s *entity.Session) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *userRepository) TouchSession(ctx context.Context, jti string) error {
	return r.db.WithContext(ctx).Model(&entity.Session{}).Where("id = ?", jti).
		Update("last_login_at", time.Now()).Error
}

func (r *userRepository) ListSessions(ctx context.Context, userID uint) ([]entity.Session, error) {
	var sessions []entity.Session
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *userRepository) DeleteSessions(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.Session{}).Error
}

func (r *userRepository) Delete(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Delete(&entity.User{}, userID).Error
}

func (r *userRepository) ListAdmin(ctx context.Context, f AdminListFilter) ([]entity.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&entity.User{})
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

func (r *userRepository) CountStats(ctx context.Context, since time.Time) (UserCounts, error) {
	var c UserCounts
	err := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("is_bot = ?", false).Count(&c.TotalUsers).Error
	if err != nil {
		return c, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("is_bot = ? AND is_active = ?", false, true).Count(&c.ActiveUsers).Error; err != nil {
		return c, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("is_bot = ? AND is_active = ?", false, false).Count(&c.BannedUsers).Error; err != nil {
		return c, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("is_bot = ? AND email_verified = ?", false, false).Count(&c.UnverifiedUsers).Error; err != nil {
		return c, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("is_bot = ?", true).Count(&c.Bots).Error; err != nil {
		return c, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("is_admin = ?", true).Count(&c.Admins).Error; err != nil {
		return c, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("is_bot = ? AND created_at >= ?", false, since).Count(&c.NewLast7Days).Error; err != nil {
		return c, err
	}
	return c, nil
}
