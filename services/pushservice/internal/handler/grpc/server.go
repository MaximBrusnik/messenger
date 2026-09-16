package grpc

import (
	"context"

	"google.golang.org/grpc/status"

	"messengermax/pkg/apperr"
	pb "messengermax/proto/gen/push"
	"messengermax/pushservice/internal/service"
)

type Server struct {
	pb.UnimplementedPushServiceServer
	svc *service.Server
}

func NewServer(svc *service.Server) *Server {
	return &Server{svc: svc}
}

func (s *Server) RegisterDevice(ctx context.Context, req *pb.RegisterDeviceRequest) (*pb.Empty, error) {
	if err := s.svc.RegisterDevice(ctx, uint(req.UserId), req.Token, req.Platform); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}

func (s *Server) UnregisterDevice(ctx context.Context, req *pb.UnregisterDeviceRequest) (*pb.Empty, error) {
	if err := s.svc.UnregisterDevice(ctx, req.Token); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}
