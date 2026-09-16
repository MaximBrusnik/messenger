package grpc

import (
	"context"

	"google.golang.org/grpc/status"

	"messengermax/pkg/apperr"
	pb "messengermax/proto/gen/realtime"
	"messengermax/realtimeservice/internal/service"
)

type Server struct {
	pb.UnimplementedRealtimeServiceServer
	svc *service.Server
}

func NewServer(svc *service.Server) *Server {
	return &Server{svc: svc}
}

func (s *Server) IsOnline(ctx context.Context, req *pb.IsOnlineRequest) (*pb.IsOnlineResponse, error) {
	online, err := s.svc.IsOnline(ctx, uint(req.UserId))
	if err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.IsOnlineResponse{Online: online}, nil
}

func (s *Server) GetOnlineStatuses(ctx context.Context, req *pb.GetOnlineStatusesRequest) (*pb.OnlineStatusesResponse, error) {
	ids := make([]uint, len(req.UserIds))
	for i, id := range req.UserIds {
		ids[i] = uint(id)
	}
	out, err := s.svc.GetOnlineStatuses(ctx, ids)
	if err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.OnlineStatusesResponse{Online: out}, nil
}

func (s *Server) Disconnect(ctx context.Context, req *pb.DisconnectRequest) (*pb.Empty, error) {
	if err := s.svc.Disconnect(ctx, uint(req.UserId)); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}
