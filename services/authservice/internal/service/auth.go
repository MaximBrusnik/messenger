package service

import (
	"context"
	"errors"
	"log"
	"time"

	"messengermax/authservice/internal/entity"
	"messengermax/authservice/internal/integration/email"
	"messengermax/authservice/internal/repo"
	"messengermax/pkg/jwt"
	pbuser "messengermax/proto/gen/user"
)

type Server struct {
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

func (s *Server) Register(ctx context.Context, username, emailAddr, password string) (*AuthResult, error) {
	if username == "" || emailAddr == "" || password == "" {
		return nil, errInvalid("все поля обязательны")
	}
	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return nil, errAlreadyExists("имя пользователя занято")
	}
	if _, err := s.userRepo.FindByEmail(emailAddr); err == nil {
		return nil, errAlreadyExists("почта уже зарегистрирована")
	}

	user := &entity.User{Username: username, Email: emailAddr}
	if err := user.HashPassword(password); err != nil {
		return nil, errInternal("не удалось сохранить пароль")
	}
	user.VerificationToken = email.GenerateVerificationToken()
	if err := s.userRepo.Create(user); err != nil {
		return nil, errInternal("не удалось создать пользователя")
	}

	s.emailSvc.SendVerificationEmail(user.Email, user.VerificationToken)

	s.syncProfile(user)
	return s.buildAuthResult(ctx, user)
}

func (s *Server) Login(ctx context.Context, username, password string) (*AuthResult, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// fell back to email lookup
			user, err = s.userRepo.FindByEmail(username)
		}
	}
	if err != nil || user == nil {
		return nil, errUnauthenticated("неверное имя пользователя или пароль")
	}
	if !user.IsActive {
		return nil, errUnauthenticated("аккаунт заблокирован")
	}
	if err := user.CheckPassword(password); err != nil {
		return nil, errUnauthenticated("неверное имя пользователя или пароль")
	}
	if s.requireEmailVerification && !user.EmailVerified {
		return nil, errPermissionDenied("подтвердите почту")
	}
	if err := s.userRepo.UpdateLastLogin(user.ID, time.Now()); err != nil {
		return nil, errInternal("не удалось обновить время входа")
	}
	return s.buildAuthResult(ctx, user)
}

func (s *Server) VerifyEmail(ctx context.Context, token string) (*AuthResult, error) {
	user, err := s.userRepo.FindByVerificationToken(token)
	if err != nil {
		return nil, errInvalid("неверный или истёкший токен")
	}
	user.EmailVerified = true
	user.VerificationToken = ""
	if err := s.userRepo.Update(user); err != nil {
		return nil, errInternal("не удалось подтвердить почту")
	}
	return s.buildAuthResult(ctx, user)
}

func (s *Server) ResendVerification(ctx context.Context, userID uint) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errNotFound("пользователь не найден")
	}
	user.VerificationToken = email.GenerateVerificationToken()
	if err := s.userRepo.Update(user); err != nil {
		return errInternal("не удалось обновить токен")
	}
	s.emailSvc.SendVerificationEmail(user.Email, user.VerificationToken)
	return nil
}

func (s *Server) Logout(ctx context.Context, userID uint) error {
	// No server-side JWT storage; logout is handled by the realtime service
	// disconnecting the WebSocket. Auth service has nothing to invalidate.
	return nil
}

func (s *Server) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return errInvalid("пароль слишком короткий")
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errNotFound("пользователь не найден")
	}
	if err := user.CheckPassword(oldPassword); err != nil {
		return errInvalid("неверный старый пароль")
	}
	if err := user.HashPassword(newPassword); err != nil {
		return errInternal("не удалось сохранить пароль")
	}
	if err := s.userRepo.UpdatePassword(user.ID, user.Password); err != nil {
		return errInternal("не удалось сохранить пароль")
	}
	return nil
}

func (s *Server) buildAuthResult(ctx context.Context, user *entity.User) (*AuthResult, error) {
	if s.requireEmailVerification && !user.EmailVerified {
		return &AuthResult{
			EmailVerificationRequired: true,
			User:                      user,
		}, nil
	}
	token, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		return nil, errInternal("не удалось выдать токен")
	}
	return &AuthResult{
		Token: token,
		User:  user,
	}, nil
}

// syncProfile mirrors a freshly created user into the userservice profile
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
