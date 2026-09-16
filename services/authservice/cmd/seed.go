package main

import (
	"context"
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
	if u, err := userRepo.FindByUsername(username); err == nil && u != nil {
		return nil
	}
	user := &entity.User{
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

	// mirror the seeded user into userservice
	if userClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		isAdmin := user.IsAdmin
		isBot := user.IsBot
		if _, err := userClient.UpdateProfile(ctx, &pbuser.UpdateProfileRequest{
			UserId:   uint64(user.ID),
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  &isAdmin,
			IsBot:    &isBot,
		}); err != nil {
			log.Printf("auth: seed sync %q failed: %v", username, err)
		}
	}
	return nil
}
