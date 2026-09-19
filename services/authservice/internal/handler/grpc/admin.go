package grpc

import (
	"context"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/authservice/internal/entity"
	"messengermax/pkg/apperr"
	pb "messengermax/proto/gen/auth"
)

func (s *Server) AdminListUsers(ctx context.Context, req *pb.AdminListUsersRequest) (*pb.AdminListUsersResponse, error) {
	list, total, err := s.admin.ListUsers(ctx, req.Query, uint(req.Page), uint(req.PageSize), req.IsBot, req.IsAdmin, req.Active)
	if err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	out := make([]*pb.AdminUser, 0, len(list))
	for i := range list {
		out = append(out, toAdminUser(&list[i]))
	}
	return &pb.AdminListUsersResponse{Users: out, Total: uint64(total)}, nil
}

func (s *Server) AdminUserStats(ctx context.Context, _ *pb.Empty) (*pb.AdminStats, error) {
	c, err := s.admin.UserStats(ctx)
	if err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.AdminStats{
		TotalUsers:      uint64(c.TotalUsers),
		ActiveUsers:     uint64(c.ActiveUsers),
		BannedUsers:     uint64(c.BannedUsers),
		UnverifiedUsers: uint64(c.UnverifiedUsers),
		Bots:            uint64(c.Bots),
		Admins:          uint64(c.Admins),
		NewLast_7Days:   uint64(c.NewLast7Days),
	}, nil
}

func (s *Server) AdminSetUserActive(ctx context.Context, req *pb.AdminSetUserActiveRequest) (*pb.Empty, error) {
	if err := s.admin.SetUserActive(ctx, uint(req.UserId), req.Active); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}

func (s *Server) AdminSetRole(ctx context.Context, req *pb.AdminSetRoleRequest) (*pb.Empty, error) {
	if err := s.admin.SetRole(ctx, uint(req.UserId), req.IsAdmin); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}

func (s *Server) AdminDeleteUser(ctx context.Context, req *pb.AdminDeleteUserRequest) (*pb.Empty, error) {
	if err := s.admin.DeleteUser(ctx, uint(req.UserId)); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}

func (s *Server) AdminRevokeSessions(ctx context.Context, req *pb.AdminRevokeSessionsRequest) (*pb.Empty, error) {
	if err := s.admin.RevokeSessions(ctx, uint(req.UserId)); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}

func (s *Server) CheckUserActive(ctx context.Context, req *pb.CheckUserActiveRequest) (*pb.CheckUserActiveResponse, error) {
	active, found, err := s.admin.CheckUserActive(ctx, uint(req.UserId))
	if err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.CheckUserActiveResponse{Active: active, Found: found}, nil
}

func toAdminUser(u *entity.User) *pb.AdminUser {
	out := &pb.AdminUser{
		Id:            uint64(u.ID),
		Username:      u.Username,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		IsBot:         u.IsBot,
		IsAdmin:       u.IsAdmin,
		IsActive:      u.IsActive,
		Status:        u.Status,
		Avatar:        u.Avatar,
		CreatedAt:     timestamppb.New(u.CreatedAt),
	}
	if !u.LastLogin.IsZero() {
		out.LastLogin = timestamppb.New(u.LastLogin)
	}
	return out
}
