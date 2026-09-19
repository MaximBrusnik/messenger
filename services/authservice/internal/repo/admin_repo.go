package repo

import (
	"time"

	"gorm.io/gorm"

	"messengermax/authservice/internal/entity"
)

type AdminListFilter struct {
	Query    string
	Page     uint
	PageSize uint
	IsBot    *bool
	IsAdmin  *bool
	Active   *bool
}

type UserCounts struct {
	TotalUsers      int64
	ActiveUsers     int64
	BannedUsers     int64
	UnverifiedUsers int64
	Bots            int64
	Admins          int64
	NewLast7Days    int64
}

type AdminRepository interface {
	FindByID(id uint) (*entity.User, error)
	Update(user *entity.User) error
	DeleteSessions(userID uint) error
	Delete(userID uint) error
	ListAdmin(f AdminListFilter) ([]entity.User, int64, error)
	CountStats(since time.Time) (UserCounts, error)
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &userRepository{db: db}
}
