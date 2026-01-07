package logic

import (
	"MessangerMax/internal/entity"
	"MessangerMax/internal/repo"
	"MessangerMax/utils"
	"errors"
	"time"
)

type AuthService interface {
	Register(req entity.RegisterRequest) (*entity.UserResponse, error)
	Login(req entity.LoginRequest) (string, *entity.UserResponse, error)
	GetUserProfile(userID uint) (*entity.UserResponse, error)
}

type authService struct {
	userRepo repo.UserRepository
	jwtUtils utils.JWTUtils
}

func NewAuthService(userRepo repo.UserRepository, jwtUtils utils.JWTUtils) AuthService {
	return &authService{
		userRepo: userRepo,
		jwtUtils: jwtUtils,
	}
}

func (s *authService) Register(req entity.RegisterRequest) (*entity.UserResponse, error) {
	// Проверка существования пользователя
	if existingUser, _ := s.userRepo.FindByUsername(req.Username); existingUser != nil {
		return nil, errors.New("пользователь с таким именем уже существует")
	}

	if existingUser, _ := s.userRepo.FindByEmail(req.Email); existingUser != nil {
		return nil, errors.New("пользователь с таким email уже существует")
	}

	// Создание пользователя
	user := &entity.User{
		Username: req.Username,
		Email:    req.Email,
		IsActive: true,
	}

	if err := user.HashPassword(req.Password); err != nil {
		return nil, err
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	response := user.ToResponse()
	return &response, nil
}

func (s *authService) Login(req entity.LoginRequest) (string, *entity.UserResponse, error) {
	// Поиск пользователя
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		return "", nil, errors.New("неверные учетные данные")
	}

	// Проверка активности
	if !user.IsActive {
		return "", nil, errors.New("учетная запись деактивирована")
	}

	// Проверка пароля
	if err := user.CheckPassword(req.Password); err != nil {
		return "", nil, errors.New("неверные учетные данные")
	}

	// Обновление времени последнего входа
	user.LastLogin = time.Now()
	s.userRepo.Update(user)

	// Генерация токена
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
