package service

import (
	"context"
	"errors"
	"log"
	"time"

	"messengermax/authservice/internal/entity"
	"messengermax/authservice/internal/repo"
	pbuser "messengermax/proto/gen/user"
)

type Admin struct {
	adminRepo  repo.AdminRepository
	userClient pbuser.UserServiceClient
}

func NewAdmin(userRepo repo.AdminRepository, userClient pbuser.UserServiceClient) *Admin {
	return &Admin{adminRepo: userRepo, userClient: userClient}
}

func (a *Admin) ListUsers(ctx context.Context, query string, page, pageSize uint, isBot, isAdmin, active *bool) ([]entity.User, int64, error) {
	list, total, err := a.adminRepo.ListAdmin(repo.AdminListFilter{
		Query:    query,
		Page:     page,
		PageSize: pageSize,
		IsBot:    isBot,
		IsAdmin:  isAdmin,
		Active:   active,
	})
	if err != nil {
		return nil, 0, errInternal("не удалось получить список пользователей")
	}
	return list, total, nil
}

func (a *Admin) UserStats(ctx context.Context) (repo.UserCounts, error) {
	return a.adminRepo.CountStats(time.Now().Add(-7 * 24 * time.Hour))
}

func (a *Admin) SetUserActive(ctx context.Context, userID uint, active bool) error {
	user, err := a.adminRepo.FindByID(userID)
	if err != nil {
		return errNotFound("пользователь не найден")
	}
	user.IsActive = active
	if err := a.adminRepo.Update(user); err != nil {
		return errInternal("не удалось обновить статус пользователя")
	}
	if !active {
		_ = a.adminRepo.DeleteSessions(userID)
	}
	a.syncProfile(user)
	return nil
}

func (a *Admin) SetRole(ctx context.Context, userID uint, isAdmin bool) error {
	user, err := a.adminRepo.FindByID(userID)
	if err != nil {
		return errNotFound("пользователь не найден")
	}
	user.IsAdmin = isAdmin
	if err := a.adminRepo.Update(user); err != nil {
		return errInternal("не удалось изменить роль")
	}
	a.syncProfile(user)
	return nil
}

func (a *Admin) DeleteUser(ctx context.Context, userID uint) error {
	user, err := a.adminRepo.FindByID(userID)
	if err != nil {
		return errNotFound("пользователь не найден")
	}
	_ = a.adminRepo.DeleteSessions(userID)
	if err := a.adminRepo.Delete(user.ID); err != nil {
		return errInternal("не удалось удалить пользователя")
	}
	return nil
}

func (a *Admin) RevokeSessions(ctx context.Context, userID uint) error {
	if err := a.adminRepo.DeleteSessions(userID); err != nil {
		return errInternal("не удалось завершить сессии")
	}
	return nil
}

func (a *Admin) CheckUserActive(ctx context.Context, userID uint) (bool, bool, error) {
	user, err := a.adminRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return false, false, nil
		}
		return false, false, err
	}
	return user.IsActive, true, nil
}

func (a *Admin) syncProfile(user *entity.User) {
	syncProfile(a.userClient, user)
}

func syncProfile(client pbuser.UserServiceClient, user *entity.User) {
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	isAdmin := user.IsAdmin
	isBot := user.IsBot
	_, err := client.UpdateProfile(ctx, &pbuser.UpdateProfileRequest{
		UserId:   uint64(user.ID),
		Username: user.Username,
		Email:    user.Email,
		IsAdmin:  &isAdmin,
		IsBot:    &isBot,
	})
	if err != nil {
		log.Printf("auth: sync profile for %d failed: %v", user.ID, err)
	}
}
