package service

import (
	"context"
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
	maxUsersEnabled          bool
	maxUsersLimit            int
	userClient               pbuser.UserServiceClient
}

func NewServer(userRepo repo.UserRepository, jwtManager *jwt.Manager, emailSvc *email.Service, requireEmailVerification bool, maxUsersEnabled bool, maxUsersLimit int, userClient pbuser.UserServiceClient) *Server {
	return &Server{
		userRepo:                 userRepo,
		jwtManager:               jwtManager,
		emailSvc:                 emailSvc,
		requireEmailVerification: requireEmailVerification,
		maxUsersEnabled:          maxUsersEnabled,
		maxUsersLimit:            maxUsersLimit,
		userClient:               userClient,
	}
}

func (s *Server) Register(ctx context.Context, username, emailAddr, password string, dev DeviceInfo) (*AuthResult, error) {
	if username == "" || emailAddr == "" || password == "" {
		return nil, errInvalid("все поля обязательны")
	}
	if _, err := s.userRepo.FindByUsername(ctx, username); err == nil {
		return nil, errAlreadyExists("имя пользователя занято")
	}
	if _, err := s.userRepo.FindByEmail(ctx, emailAddr); err == nil {
		return nil, errAlreadyExists("почта уже зарегистрирована")
	}

	if s.maxUsersEnabled {
		count, err := s.userRepo.CountUsers(ctx)
		if err != nil {
			return nil, errInternal("не удалось проверить количество пользователей")
		}
		if count >= int64(s.maxUsersLimit) {
			return nil, errResourceExhausted("достигнут лимит пользователей")
		}
	}

	user := &entity.User{Username: username, Email: emailAddr}
	if err := user.HashPassword(password); err != nil {
		return nil, errInternal("не удалось сохранить пароль")
	}
	user.VerificationToken = email.GenerateVerificationToken()
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errInternal("не удалось создать пользователя")
	}

	s.emailSvc.SendVerificationEmail(user.Email, user.VerificationToken)

	s.syncProfile(user)
	return s.buildAuthResult(ctx, user, dev)
}

func (s *Server) Login(ctx context.Context, email, password string, dev DeviceInfo) (*AuthResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// legacy: администратор входит по username
		user, err = s.adminByUsername(ctx, email)
	}
	if err != nil || user == nil {
		return nil, errUnauthenticated("неверная почта или пароль")
	}
	if !user.IsActive {
		return nil, errUnauthenticated("аккаунт заблокирован")
	}
	if err := user.CheckPassword(password); err != nil {
		return nil, errUnauthenticated("неверная почта или пароль")
	}
	if s.requireEmailVerification && !user.EmailVerified {
		return nil, errPermissionDenied("подтвердите почту")
	}
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID, time.Now()); err != nil {
		return nil, errInternal("не удалось обновить время входа")
	}
	return s.buildAuthResult(ctx, user, dev)
}

func (s *Server) VerifyEmail(ctx context.Context, token string, dev DeviceInfo) (*AuthResult, error) {
	user, err := s.userRepo.FindByVerificationToken(ctx, token)
	if err != nil {
		return nil, errInvalid("неверный или истёкший токен")
	}
	user.EmailVerified = true
	user.VerificationToken = ""
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, errInternal("не удалось подтвердить почту")
	}
	return s.buildAuthResult(ctx, user, dev)
}

func (s *Server) ResendVerification(ctx context.Context, userID uint) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errNotFound("пользователь не найден")
	}
	user.VerificationToken = email.GenerateVerificationToken()
	if err := s.userRepo.Update(ctx, user); err != nil {
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
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errNotFound("пользователь не найден")
	}
	if err := user.CheckPassword(oldPassword); err != nil {
		return errInvalid("неверный старый пароль")
	}
	if err := user.HashPassword(newPassword); err != nil {
		return errInternal("не удалось сохранить пароль")
	}
	if err := s.userRepo.UpdatePassword(ctx, user.ID, user.Password); err != nil {
		return errInternal("не удалось сохранить пароль")
	}
	return nil
}

func (s *Server) adminByUsername(ctx context.Context, username string) (*entity.User, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if !user.IsAdmin {
		return nil, repo.ErrNotFound
	}
	return user, nil
}

func (s *Server) buildAuthResult(ctx context.Context, user *entity.User, dev DeviceInfo) (*AuthResult, error) {
	if s.requireEmailVerification && !user.EmailVerified {
		return &AuthResult{
			EmailVerificationRequired: true,
			User:                      user,
		}, nil
	}
	token, jti, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		return nil, errInternal("не удалось выдать токен")
	}
	if err := s.userRepo.CreateSession(ctx, &entity.Session{
		ID:                jti,
		UserID:            user.ID,
		DeviceName:        dev.Name,
		Platform:          dev.Platform,
		DeviceFingerprint: dev.Fingerprint,
		IP:                dev.IP,
		TokenExpiresAt:    time.Now().Add(24 * time.Hour),
		LastLoginAt:       time.Now(),
	}); err != nil {
		log.Printf("auth: record session for %d failed: %v", user.ID, err)
	}
	return &AuthResult{
		Token: token,
		User:  user,
	}, nil
}

func (s *Server) ListDevices(ctx context.Context, userID uint, currentJTI string) ([]DeviceSummary, error) {
	sessions, err := s.userRepo.ListSessions(ctx, userID)
	if err != nil {
		return nil, errInternal("не удалось получить список устройств")
	}
	if currentJTI != "" {
		_ = s.userRepo.TouchSession(ctx, currentJTI)
	}
	currentFP := ""
	for _, sess := range sessions {
		if sess.ID == currentJTI {
			currentFP = sess.DeviceFingerprint
			break
		}
	}
	groups := make(map[string]*DeviceSummary, len(sessions))
	order := make([]string, 0, len(sessions))
	for _, sess := range sessions {
		fp := sess.DeviceFingerprint
		if fp == "" {
			fp = "unknown"
		}
		g, ok := groups[fp]
		if !ok {
			g = &DeviceSummary{
				Name:       sess.DeviceName,
				Platform:   sess.Platform,
				FirstLogin: sess.CreatedAt,
				LastLogin:  sess.LastLoginAt,
			}
			if g.LastLogin.IsZero() {
				g.LastLogin = sess.CreatedAt
			}
			groups[fp] = g
			order = append(order, fp)
		}
		g.LoginCount++
		if sess.CreatedAt.Before(g.FirstLogin) {
			g.FirstLogin = sess.CreatedAt
		}
		if sess.LastLoginAt.After(g.LastLogin) {
			g.LastLogin = sess.LastLoginAt
		}
		if g.IP == "" {
			g.IP = sess.IP
		}
		if fp == currentFP {
			g.IsCurrent = true
		}
	}
	out := make([]DeviceSummary, 0, len(order))
	for _, fp := range order {
		out = append(out, *groups[fp])
	}
	return out, nil
}

func (s *Server) syncProfile(user *entity.User) {
	syncProfile(s.userClient, user)
}
