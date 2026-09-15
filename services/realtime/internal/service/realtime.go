package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"messengermax/pkg/redis"
	pb "messengermax/proto/gen/realtime"

	"messengermax/realtime/internal/ws"
)

// Server implements the RealtimeService gRPC contract. Online state is
// served from Redis (written by the WS handler); presence in the hub is
// used to force a disconnect.
type Server struct {
	pb.UnimplementedRealtimeServiceServer
	redis  *redis.Client
	wsCtrl *ws.Controller
}

func NewServer(redis *redis.Client, wsCtrl *ws.Controller) *Server {
	return &Server{redis: redis, wsCtrl: wsCtrl}
}

func (s *Server) IsOnline(ctx context.Context, req *pb.IsOnlineRequest) (*pb.IsOnlineResponse, error) {
	return &pb.IsOnlineResponse{Online: s.redis.IsOnline(ctx, uint(req.UserId))}, nil
}

func (s *Server) GetOnlineStatuses(ctx context.Context, req *pb.GetOnlineStatusesRequest) (*pb.OnlineStatusesResponse, error) {
	out := &pb.OnlineStatusesResponse{Online: make(map[uint64]bool, len(req.UserIds))}
	for _, id := range req.UserIds {
		out.Online[id] = s.redis.IsOnline(ctx, uint(id))
	}
	return out, nil
}

func (s *Server) Disconnect(ctx context.Context, req *pb.DisconnectRequest) (*pb.Empty, error) {
	if err := s.wsCtrl.DisconnectUser(ctx, uint(req.UserId)); err != nil {
		return nil, status.Error(codes.Internal, "failed to disconnect user")
	}
	return &pb.Empty{}, nil
}
