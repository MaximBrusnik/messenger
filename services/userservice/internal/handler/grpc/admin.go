package grpc

import (
	"context"

	pb "messengermax/proto/gen/user"
)

func (s *Server) AdminOnlineCount(ctx context.Context, _ *pb.Empty) (*pb.AdminOnlineCountResponse, error) {
	n, err := s.admin.OnlineCount(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.AdminOnlineCountResponse{OnlineUsers: uint64(n)}, nil
}

func (s *Server) AdminDeleteProfile(ctx context.Context, req *pb.AdminDeleteProfileRequest) (*pb.Empty, error) {
	if err := s.admin.DeleteProfile(ctx, uint(req.UserId)); err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.Empty{}, nil
}
