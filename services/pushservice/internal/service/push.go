package service

import (
	"context"

	"messengermax/pushservice/internal/entity"
	"messengermax/pushservice/internal/repo"
)

type Server struct {
	deviceRepo repo.DeviceTokenRepository
}

func NewServer(deviceRepo repo.DeviceTokenRepository) *Server {
	return &Server{deviceRepo: deviceRepo}
}

func (s *Server) RegisterDevice(ctx context.Context, userID uint, token, platform string) error {
	if token == "" {
		return errInvalid("token required")
	}
	_ = s.deviceRepo.DeleteByToken(token)
	t := &entity.DeviceToken{UserID: userID, Token: token, Platform: platform}
	if err := s.deviceRepo.Add(t); err != nil {
		return errInternal("failed to register device")
	}
	return nil
}

func (s *Server) UnregisterDevice(ctx context.Context, token string) error {
	if err := s.deviceRepo.DeleteByToken(token); err != nil {
		return errInternal("failed to unregister device")
	}
	return nil
}
