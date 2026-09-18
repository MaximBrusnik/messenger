package main

import (
	"context"
	"errors"
	"log"
	"time"

	"messengermax/authservice/internal/entity"
	"messengermax/authservice/internal/repo"
	pbuser "messengermax/proto/gen/user"
)

func seedDefaults(userRepo repo.UserRepository, userClient pbuser.UserServiceClient) error {
	var err error
	if err = seedIfMissing(userRepo, userClient, "Ассистент", "ai@messengermax.local", "", true, false); err != nil {
		return err
	}
	return seedIfMissing(userRepo, userClient, "Admin", "admin@messengermax.local", "07062002", false, true)
}

func seedIfMissing(userRepo repo.UserRepository, userClient pbuser.UserServiceClient, username, email, password string, isBot, isAdmin bool) error {
	user, err := userRepo.FindByUsername(username)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return err
	}
	if err != nil {
		user = &entity.User{
			Username:      username,
			Email:         email,
			IsActive:      true,
			IsBot:         isBot,
			IsAdmin:       isAdmin,
			EmailVerified: true,
		}
		pw := password
		if pw == "" {
			pw = "none"
		}
		if err := user.HashPassword(pw); err != nil {
			return err
		}
		if err := userRepo.Create(user); err != nil {
			return err
		}
		log.Printf("auth: seeded %q", username)
	}

	if userClient == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	adminFlag := user.IsAdmin
	botFlag := user.IsBot
	if _, err := userClient.UpdateProfile(ctx, &pbuser.UpdateProfileRequest{
		UserId:   uint64(user.ID),
		Username: user.Username,
		Email:    user.Email,
		IsAdmin:  &adminFlag,
		IsBot:    &botFlag,
	}); err != nil {
		log.Printf("auth: seed sync %q failed: %v", username, err)
		return err
	}
	return nil
}
