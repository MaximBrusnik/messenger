package logic

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"MessangerMax/utils"
	"errors"
	"log"
	"time"
)

type AuthService interface {
	Register(req entity.RegisterRequest) (*entity.UserResponse, error)
	Login(req entity.LoginRequest) (string, *entity.UserResponse, error)
	GetUserProfile(userID uint) (*entity.UserResponse, error)
	VerifyEmail(token string) error
	ResendVerification(userID uint) error
}

type authService struct {
	userRepo     repo.UserRepository
	jwtUtils     utils.JWTUtils
	emailService EmailService
}

func NewAuthService(userRepo repo.UserRepository, jwtUtils utils.JWTUtils, emailService EmailService) AuthService {
	return &authService{
		userRepo:     userRepo,
		jwtUtils:     jwtUtils,
		emailService: emailService,
	}
}

func (s *authService) Register(req entity.RegisterRequest) (*entity.UserResponse, error) {
	if existingUser, _ := s.userRepo.FindByUsername(req.Username); existingUser != nil {
		return nil, errors.New("пользователь с таким именем уже существует")
	}

	if existingUser, _ := s.userRepo.FindByEmail(req.Email); existingUser != nil {
		return nil, errors.New("пользователь с таким email уже существует")
	}

	user := &entity.User{
		Username:          req.Username,
		Email:             req.Email,
		IsActive:          true,
		VerificationToken: GenerateVerificationToken(),
	}

	if err := user.HashPassword(req.Password); err != nil {
		return nil, err
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	if err := s.emailService.SendVerificationEmail(user.Email, user.VerificationToken); err != nil {
		log.Printf("Warning: failed to send verification email: %v", err)
	}

	response := user.ToResponse()
	return &response, nil
}

func (s *authService) Login(req entity.LoginRequest) (string, *entity.UserResponse, error) {
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		return "", nil, errors.New("неверные учетные данные")
	}

	if !user.IsActive {
		return "", nil, errors.New("учетная запись деактивирована")
	}

	if err := user.CheckPassword(req.Password); err != nil {
		return "", nil, errors.New("неверные учетные данные")
	}

	user.LastLogin = time.Now()
	s.userRepo.Update(user)

	token, err := s.jwtUtils.GenerateToken(user.ID)
	if err != nil {
		return "", nil, err
	}

	response := user.ToResponse()
	return token, &response, nil
}

func (s *authService) GetUserProfile(userID uint) (*entity.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("пользователь не найден")
	}

	response := user.ToResponse()
	return &response, nil
}

func (s *authService) VerifyEmail(token string) error {
	user, err := s.userRepo.FindByVerificationToken(token)
	if err != nil {
		return errors.New("неверный или истёкший токен")
	}

	user.EmailVerified = true
	user.VerificationToken = ""

	return s.userRepo.Update(user)
}

func (s *authService) ResendVerification(userID uint) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("пользователь не найден")
	}

	if user.EmailVerified {
		return errors.New("email уже подтверждён")
	}

	user.VerificationToken = GenerateVerificationToken()
	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	return s.emailService.SendVerificationEmail(user.Email, user.VerificationToken)
}
