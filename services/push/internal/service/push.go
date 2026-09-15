package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "messengermax/proto/gen/push"
	"messengermax/push/internal/entity"
	"messengermax/push/internal/repo"
)

// Server implements push device registration.
type Server struct {
	pb.UnimplementedPushServiceServer
	deviceRepo repo.DeviceTokenRepository
}

func NewServer(deviceRepo repo.DeviceTokenRepository) *Server {
	return &Server{deviceRepo: deviceRepo}
}

func (s *Server) RegisterDevice(ctx context.Context, req *pb.RegisterDeviceRequest) (*pb.Empty, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token required")
	}
	_ = s.deviceRepo.DeleteByToken(req.Token)
	t := &entity.DeviceToken{UserID: uint(req.UserId), Token: req.Token, Platform: req.Platform}
	if err := s.deviceRepo.Add(t); err != nil {
		return nil, status.Error(codes.Internal, "failed to register device")
	}
	return &pb.Empty{}, nil
}

func (s *Server) UnregisterDevice(ctx context.Context, req *pb.UnregisterDeviceRequest) (*pb.Empty, error) {
	if err := s.deviceRepo.DeleteByToken(req.Token); err != nil {
		return nil, status.Error(codes.Internal, "failed to unregister device")
	}
	return &pb.Empty{}, nil
}
