package service

import (
	"context"
	"errors"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"messengermax/auth/internal/email"
	"messengermax/auth/internal/entity"
	"messengermax/auth/internal/repo"
	"messengermax/pkg/jwt"
	pb "messengermax/proto/gen/auth"
	pbuser "messengermax/proto/gen/user"
)

type Server struct {
	pb.UnimplementedAuthServiceServer
	userRepo                 repo.UserRepository
	jwtManager               *jwt.Manager
	emailSvc                 *email.Service
	requireEmailVerification bool
	userClient               pbuser.UserServiceClient
}

func NewServer(userRepo repo.UserRepository, jwtManager *jwt.Manager, emailSvc *email.Service, requireEmailVerification bool, userClient pbuser.UserServiceClient) *Server {
	return &Server{
		userRepo:                 userRepo,
		jwtManager:               jwtManager,
		emailSvc:                 emailSvc,
		requireEmailVerification: requireEmailVerification,
		userClient:               userClient,
	}
}

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "все поля обязательны")
	}
	if _, err := s.userRepo.FindByUsername(req.Username); err == nil {
		return nil, status.Error(codes.AlreadyExists, "имя пользователя занято")
	}
	if _, err := s.userRepo.FindByEmail(req.Email); err == nil {
		return nil, status.Error(codes.AlreadyExists, "почта уже зарегистрирована")
	}

	user := &entity.User{Username: req.Username, Email: req.Email}
	if err := user.HashPassword(req.Password); err != nil {
		return nil, status.Error(codes.Internal, "не удалось сохранить пароль")
	}
	user.VerificationToken = email.GenerateVerificationToken()
	if err := s.userRepo.Create(user); err != nil {
		return nil, status.Error(codes.Internal, "не удалось создать пользователя")
	}

	s.emailSvc.SendVerificationEmail(user.Email, user.VerificationToken)

	s.syncProfile(user)
	return s.buildAuthResponse(ctx, user)
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// fell back to email lookup
			user, err = s.userRepo.FindByEmail(req.Username)
		}
	}
	if err != nil || user == nil {
		return nil, status.Error(codes.Unauthenticated, "неверное имя пользователя или пароль")
	}
	if !user.IsActive {
		return nil, status.Error(codes.Unauthenticated, "аккаунт заблокирован")
	}
	if err := user.CheckPassword(req.Password); err != nil {
		return nil, status.Error(codes.Unauthenticated, "неверное имя пользователя или пароль")
	}
	if s.requireEmailVerification && !user.EmailVerified {
		return nil, status.Error(codes.PermissionDenied, "подтвердите почту")
	}
	if err := s.userRepo.UpdateLastLogin(user.ID, time.Now()); err != nil {
		return nil, status.Error(codes.Internal, "не удалось обновить время входа")
	}
	return s.buildAuthResponse(ctx, user)
}

func (s *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.AuthResponse, error) {
	user, err := s.userRepo.FindByVerificationToken(req.Token)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "неверный или истёкший токен")
	}
	user.EmailVerified = true
	user.VerificationToken = ""
	if err := s.userRepo.Update(user); err != nil {
		return nil, status.Error(codes.Internal, "не удалось подтвердить почту")
	}
	return s.buildAuthResponse(ctx, user)
}

func (s *Server) ResendVerification(ctx context.Context, req *pb.ResendVerificationRequest) (*pb.Empty, error) {
	user, err := s.userRepo.FindByID(uint(req.UserId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "пользователь не найден")
	}
	user.VerificationToken = email.GenerateVerificationToken()
	if err := s.userRepo.Update(user); err != nil {
		return nil, status.Error(codes.Internal, "не удалось обновить токен")
	}
	s.emailSvc.SendVerificationEmail(user.Email, user.VerificationToken)
	return &pb.Empty{}, nil
}

func (s *Server) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.Empty, error) {
	// No server-side JWT storage; logout is handled by the realtime service
	// disconnecting the WebSocket. Auth service has nothing to invalidate.
	return &pb.Empty{}, nil
}

func (s *Server) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.Empty, error) {
	if len(req.NewPassword) < 6 {
		return nil, status.Error(codes.InvalidArgument, "пароль слишком короткий")
	}
	user, err := s.userRepo.FindByID(uint(req.UserId))
	if err != nil {
		return nil, status.Error(codes.NotFound, "пользователь не найден")
	}
	if err := user.CheckPassword(req.OldPassword); err != nil {
		return nil, status.Error(codes.InvalidArgument, "неверный старый пароль")
	}
	if err := user.HashPassword(req.NewPassword); err != nil {
		return nil, status.Error(codes.Internal, "не удалось сохранить пароль")
	}
	if err := s.userRepo.UpdatePassword(user.ID, user.Password); err != nil {
		return nil, status.Error(codes.Internal, "не удалось сохранить пароль")
	}
	return &pb.Empty{}, nil
}

func (s *Server) buildAuthResponse(ctx context.Context, user *entity.User) (*pb.AuthResponse, error) {
	if s.requireEmailVerification && !user.EmailVerified {
		return &pb.AuthResponse{
			EmailVerificationRequired: true,
			User:                      toProto(user),
		}, nil
	}
	token, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось выдать токен")
	}
	return &pb.AuthResponse{
		Token: token,
		User:  toProto(user),
	}, nil
}

// syncProfile mirrors a freshly created user into the user-service profile
// store so the messenger can display it. Failures are non-fatal.
func (s *Server) syncProfile(user *entity.User) {
	if s.userClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	isAdmin := user.IsAdmin
	isBot := user.IsBot
	_, err := s.userClient.UpdateProfile(ctx, &pbuser.UpdateProfileRequest{
		UserId:   uint64(user.ID),
		Username: user.Username,
		Email:    user.Email,
		IsAdmin:  &isAdmin,
		IsBot:    &isBot,
	})
	if err != nil {
		log.Printf("auth: sync profile for %d failed: %v", user.ID, err)
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
