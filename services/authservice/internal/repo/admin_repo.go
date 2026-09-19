package repo

import (
	"context"
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
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	DeleteSessions(ctx context.Context, userID uint) error
	Delete(ctx context.Context, userID uint) error
	ListAdmin(ctx context.Context, f AdminListFilter) ([]entity.User, int64, error)
	CountStats(ctx context.Context, since time.Time) (UserCounts, error)
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &userRepository{db: db}
}
