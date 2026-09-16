package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/authservice/internal/entity"
	"messengermax/authservice/internal/service"
	"messengermax/pkg/apperr"
	pb "messengermax/proto/gen/auth"
)

type Server struct {
	pb.UnimplementedAuthServiceServer
	svc *service.Server
}

func NewServer(svc *service.Server) *Server {
	return &Server{svc: svc}
}

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	res, err := s.svc.Register(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return toAuthResponse(res), nil
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	res, err := s.svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return toAuthResponse(res), nil
}

func (s *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.AuthResponse, error) {
	res, err := s.svc.VerifyEmail(ctx, req.Token)
	if err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return toAuthResponse(res), nil
}

func (s *Server) ResendVerification(ctx context.Context, req *pb.ResendVerificationRequest) (*pb.Empty, error) {
	if err := s.svc.ResendVerification(ctx, uint(req.UserId)); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}

func (s *Server) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.Empty, error) {
	if err := s.svc.Logout(ctx, uint(req.UserId)); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}

func (s *Server) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.Empty, error) {
	if err := s.svc.ChangePassword(ctx, uint(req.UserId), req.OldPassword, req.NewPassword); err != nil {
		return nil, status.Error(apperr.CodeOf(err), apperr.MsgOf(err))
	}
	return &pb.Empty{}, nil
}

func toAuthResponse(res *service.AuthResult) *pb.AuthResponse {
	if res.EmailVerificationRequired {
		return &pb.AuthResponse{
			EmailVerificationRequired: true,
			User:                      toProto(res.User),
		}
	}
	return &pb.AuthResponse{
		Token: res.Token,
		User:  toProto(res.User),
	}
}

func toProto(u *entity.User) *pb.User {
	return &pb.User{
		Id:            uint64(u.ID),
		Username:      u.Username,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		IsBot:         u.IsBot,
		IsAdmin:       u.IsAdmin,
		Status:        u.Status,
		Avatar:        u.Avatar,
		CreatedAt:     timestamppb.New(u.CreatedAt),
		LastLogin:     timestampOrNil(u.LastLogin),
	}
}

func timestampOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
