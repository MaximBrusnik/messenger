package service

import (
	"context"

	sharedredis "messengermax/pkg/redis"

	"messengermax/userservice/internal/repo"
)

type Admin struct {
	profileRepo repo.AdminRepository
	redis       *sharedredis.Client
}

func NewAdmin(profileRepo repo.AdminRepository, redis *sharedredis.Client) *Admin {
	return &Admin{profileRepo: profileRepo, redis: redis}
}

func (a *Admin) OnlineCount(ctx context.Context) (int64, error) {
	if a.redis == nil {
		return 0, errUnavailable("redis unavailable")
	}
	n, err := a.redis.CountOnline(ctx)
	if err != nil {
		return 0, errInternal("не удалось подсчитать онлайн-пользователей")
	}
	return n, nil
}

func (a *Admin) DeleteProfile(ctx context.Context, userID uint) error {
	if err := a.profileRepo.Delete(ctx, userID); err != nil {
		return errInternal("не удалось удалить профиль")
	}
	if a.redis != nil {
		_ = a.redis.DeleteCachedProfile(ctx, userID)
	}
	return nil
}
